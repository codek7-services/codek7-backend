// Compile with:
// g++ -g quality_transcode.cpp -o quality_transcode `pkg-config --cflags --libs libavformat libavcodec libavutil libswscale`

extern "C" {
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libswscale/swscale.h>
#include <libavutil/imgutils.h>
#include <libavutil/opt.h>
}

#include <iostream>
#include <vector>
#include <string>

struct Output {
    std::string filename;
    std::string crf;
    AVCodecContext* enc_ctx = nullptr;
    AVFormatContext* fmt_ctx = nullptr;
    AVStream* stream = nullptr;
    AVFrame* frame = nullptr;
};

bool setup_output(Output& out, int width, int height, AVCodecID codec_id, const AVPixelFormat pix_fmt, AVRational time_base) {
    const AVCodec* encoder = avcodec_find_encoder(codec_id);
    if (!encoder) return false;

    out.enc_ctx = avcodec_alloc_context3(encoder);
    out.enc_ctx->width = width;
    out.enc_ctx->height = height;
    out.enc_ctx->pix_fmt = pix_fmt;
    out.enc_ctx->time_base = time_base;

    // Set encoder options
    av_opt_set(out.enc_ctx->priv_data, "crf", out.crf.c_str(), 0);
    av_opt_set(out.enc_ctx->priv_data, "preset", "slow", 0);
    av_opt_set(out.enc_ctx->priv_data, "profile", "high", 0);
    av_opt_set(out.enc_ctx->priv_data, "tune", "film", 0);

    if (avcodec_open2(out.enc_ctx, encoder, nullptr) < 0) return false;

    avformat_alloc_output_context2(&out.fmt_ctx, nullptr, nullptr, out.filename.c_str());
    if (!out.fmt_ctx) return false;

    out.stream = avformat_new_stream(out.fmt_ctx, nullptr);
    avcodec_parameters_from_context(out.stream->codecpar, out.enc_ctx);
    out.stream->time_base = out.enc_ctx->time_base;

    if (avio_open(&out.fmt_ctx->pb, out.filename.c_str(), AVIO_FLAG_WRITE) < 0) return false;
    avformat_write_header(out.fmt_ctx, nullptr);

    out.frame = av_frame_alloc();
    av_image_alloc(out.frame->data, out.frame->linesize, width, height, pix_fmt, 32);
    out.frame->width = width;
    out.frame->height = height;
    out.frame->format = pix_fmt;

    return true;
}

void cleanup_output(Output& out) {
    if (out.frame) {
        av_freep(&out.frame->data[0]);
        av_frame_free(&out.frame);
    }
    if (out.enc_ctx) avcodec_free_context(&out.enc_ctx);
    if (out.fmt_ctx) {
        av_write_trailer(out.fmt_ctx);
        avio_closep(&out.fmt_ctx->pb);
        avformat_free_context(out.fmt_ctx);
    }
}

int main() {
    const char* input_file = "video.mp4";

    AVFormatContext* in_fmt = nullptr;
    if (avformat_open_input(&in_fmt, input_file, nullptr, nullptr) < 0 ||
        avformat_find_stream_info(in_fmt, nullptr) < 0) {
        std::cerr << "Failed to open input\n"; return 1;
    }

    int video_index = -1;
    for (unsigned i = 0; i < in_fmt->nb_streams; ++i) {
        if (in_fmt->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
            video_index = i;
            break;
        }
    }

    if (video_index == -1) {
        std::cerr << "No video stream found\n"; return 1;
    }

    const AVCodec* decoder = avcodec_find_decoder(in_fmt->streams[video_index]->codecpar->codec_id);
    AVCodecContext* dec_ctx = avcodec_alloc_context3(decoder);
    avcodec_parameters_to_context(dec_ctx, in_fmt->streams[video_index]->codecpar);
    if (avcodec_open2(dec_ctx, decoder, nullptr) < 0) {
        std::cerr << "Failed to open decoder\n"; return 1;
    }

    int width = dec_ctx->width;
    int height = dec_ctx->height;

    // Define outputs with same resolution but different CRFs
    std::vector<Output> outputs = {
        {"out_crf18.mp4", "18"},
        {"out_crf23.mp4", "23"},
        {"out_crf28.mp4", "28"},
    };

    for (auto& out : outputs) {
        if (!setup_output(out, width, height, AV_CODEC_ID_H264, AV_PIX_FMT_YUV420P, {1, 25})) {
            std::cerr << "Failed to setup output " << out.filename << "\n";
            return 1;
        }
    }

    AVFrame* in_frame = av_frame_alloc();
    AVPacket* pkt = av_packet_alloc();
    AVPacket* out_pkt = av_packet_alloc();

    int frame_index = 0;
    while (av_read_frame(in_fmt, pkt) >= 0) {
        if (pkt->stream_index != video_index) {
            av_packet_unref(pkt);
            continue;
        }

        if (avcodec_send_packet(dec_ctx, pkt) < 0) {
            av_packet_unref(pkt);
            continue;
        }

        while (avcodec_receive_frame(dec_ctx, in_frame) == 0) {
            for (auto& out : outputs) {
                av_frame_copy(out.frame, in_frame);
                av_frame_copy_props(out.frame, in_frame);
                out.frame->pts = frame_index;

                if (avcodec_send_frame(out.enc_ctx, out.frame) >= 0) {
                    while (avcodec_receive_packet(out.enc_ctx, out_pkt) == 0) {
                        av_packet_rescale_ts(out_pkt, out.enc_ctx->time_base, out.stream->time_base);
                        out_pkt->stream_index = out.stream->index;
                        av_interleaved_write_frame(out.fmt_ctx, out_pkt);
                        av_packet_unref(out_pkt);
                    }
                }
            }
            frame_index++;
        }

        av_packet_unref(pkt);
    }

    // Flush encoders
    for (auto& out : outputs) {
        avcodec_send_frame(out.enc_ctx, nullptr);
        while (avcodec_receive_packet(out.enc_ctx, out_pkt) == 0) {
            av_packet_rescale_ts(out_pkt, out.enc_ctx->time_base, out.stream->time_base);
            out_pkt->stream_index = out.stream->index;
            av_interleaved_write_frame(out.fmt_ctx, out_pkt);
            av_packet_unref(out_pkt);
        }
    }

    // Cleanup
    for (auto& out : outputs) cleanup_output(out);
    av_packet_free(&pkt);
    av_packet_free(&out_pkt);
    av_frame_free(&in_frame);
    avcodec_free_context(&dec_ctx);
    avformat_close_input(&in_fmt);

    std::cout << "✅ Same-resolution videos saved with different CRF quality settings.\n";
    return 0;
}


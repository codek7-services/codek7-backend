use crate::consts::{NSFW_RESOLUTIONS, RESOLUTIONS};
use crate::repo::{upload_video_request::Data, UploadVideoRequest, VideoChunk, VideoMetadata};
use crate::video::{generate_resolutions, generate_segments, save_video};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::Headers;
use rdkafka::message::Message;
use rdkafka::util::get_rdkafka_version;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio::fs::File;
use tokio::io::{AsyncReadExt, BufReader};
use tokio::sync::mpsc;
use tokio_stream::wrappers::ReceiverStream;
use tokio_stream::StreamExt;
use tonic::Request;

type ChunkIndex = usize;
type TotalChunks = usize;

pub async fn consume_video_chunks(rpc_client: crate::rpc::RpcClient, rmq: crate::rmq::RabbitMQ) {
    let (version_n, version_s) = get_rdkafka_version();
    println!("🌀 rdkafka version: 0x{:08x}, {}", version_n, version_s);

    let consumer: StreamConsumer = ClientConfig::new()
        .set("bootstrap.servers", "localhost:9092")
        .set("group.id", "video-consumer-group")
        .set("auto.offset.reset", "earliest")
        .create()
        .expect("Consumer creation failed");

    consumer
        .subscribe(&["video-chunks"])
        .expect("Subscribe failed");

    println!("🚀 Listening for video chunks...");

    let mut stream = consumer.stream();

    // Thread-safe state to store received chunks per video_id
    let video_map: Arc<Mutex<HashMap<String, (HashMap<ChunkIndex, Vec<u8>>, TotalChunks)>>> =
        Arc::new(Mutex::new(HashMap::new()));

    while let Some(result) = stream.next().await {
        match result {
            Ok(msg) => {
                let video_id = msg.key().map(|k| String::from_utf8_lossy(k).to_string());
                let payload = msg.payload().map(|p| p.to_vec());

                if video_id.is_none() || payload.is_none() {
                    continue;
                }

                let video_id = video_id.unwrap();
                let payload = payload.unwrap();

                let mut chunk_index: Option<usize> = None;
                let mut total_chunks: Option<usize> = None;
                let mut file_path: Option<usize> = None;
                let mut title: Option<String> = None;
                let mut user_id: Option<String> = None;
                let mut description: Option<String> = None;

                if let Some(headers) = msg.headers() {
                    for i in 0..headers.count() {
                        let header = headers.get(i);
                        match header.key {
                            "chunk_index" => {
                                if let Some(val) = header.value {
                                    chunk_index = String::from_utf8_lossy(val).parse().ok();
                                }
                            }
                            "total_chunks" => {
                                if let Some(val) = header.value {
                                    total_chunks = String::from_utf8_lossy(val).parse().ok();
                                }
                            }
                            "file_path" => {
                                if let Some(val) = header.value {
                                    file_path = String::from_utf8_lossy(val).parse().ok();
                                }
                            }
                            "user_id" => {
                                if let Some(val) = header.value {
                                    user_id = String::from_utf8_lossy(val).parse().ok();

                                    println!("user_id: {:?}", user_id);
                                }
                            }
                            "title" => {
                                if let Some(val) = header.value {
                                    title = String::from_utf8_lossy(val).parse().ok();
                                    println!("title: {:?}", title);
                                }
                            }
                            "description" => {
                                if let Some(val) = header.value {
                                    description = String::from_utf8_lossy(val).parse().ok();
                                    println!("description: {:?}", description);
                                }
                            }
                            _ => {}
                        }
                    }
                }

                if let (Some(index), Some(total)) = (chunk_index, total_chunks) {
                    let mut map = video_map.lock().unwrap();
                    let entry = map
                        .entry(video_id.clone())
                        .or_insert_with(|| (HashMap::new(), total));
                    entry.0.insert(index, payload);

                    if entry.0.len() == total {
                        println!("✅ Received all chunks for video '{}'. Saving...", video_id);
                        save_video(&video_id, &entry.0).await;
                        let mut file_paths = generate_segments(
                            format!("{}.mp4", &video_id).as_str(),
                            &video_id,
                            RESOLUTIONS,
                        );
                        let to_nsfw_path = generate_resolutions(
                            format!("{}.mp4", &video_id).as_str(),
                            &video_id,
                            NSFW_RESOLUTIONS,
                        );
                        file_paths.extend(to_nsfw_path.clone());
                        let rpc = rpc_client.get_client();

                        println!("📦 Uploading generated files...");

                        println!("📦 Uploading generated resolution files...");

                        // Replace the problematic section in your code with this:

                        // Fixed version of your upload loop
                        for path in &file_paths {
                            let file_name = path;

                            println!("📦 Uploading file: {}", file_name);

                            let file_size = tokio::fs::metadata(path)
                                .await
                                .map(|m| m.len())
                                .unwrap_or(0);

                            let title_ = format!("{}", title.as_deref().unwrap_or("Untitled"),);
                            let user_id_ = user_id.as_deref().unwrap_or("unknown").to_string();
                            let description_ = description.as_deref().unwrap_or("").to_string();

                            // Create a NEW channel for EACH file
                            let (tx, rx) = mpsc::channel(16); // Increased buffer size

                            // Send metadata for this specific file
                            println!("➡️ Sending metadata for {}", file_name);
                            let metadata = UploadVideoRequest {
                                data: Some(Data::Metadata(VideoMetadata {
                                    user_id: user_id_.to_string(),
                                    title: title_.to_string(),
                                    description: description_.to_string(),
                                    file_name: file_name.to_string(),
                                    file_size: file_size as i64,
                                })),
                            };

                            if let Err(e) = tx.send(metadata).await {
                                eprintln!("❌ Failed to send metadata for {}: {}", file_name, e);
                                continue;
                            }

                            // Send all chunks for this file
                            let upload_result = {
                                let file = match File::open(path).await {
                                    Ok(f) => f,
                                    Err(e) => {
                                        eprintln!("❌ Could not open {}: {}", path, e);
                                        drop(tx);
                                        continue;
                                    }
                                };

                                let mut reader = BufReader::new(file);
                                let mut buffer = vec![0u8; 1024 * 1024];
                                let mut chunk_number = 0;
                                let mut chunk_send_success = true;

                                loop {
                                    let bytes_read = match reader.read(&mut buffer).await {
                                        Ok(0) => break, // EOF
                                        Ok(n) => n,
                                        Err(e) => {
                                            eprintln!(
                                                "❌ Failed to read chunk from {}: {}",
                                                file_name, e
                                            );
                                            chunk_send_success = false;
                                            break;
                                        }
                                    };

                                    println!(
                                        "📦 Reading chunk {} from {} ({} bytes)",
                                        chunk_number, file_name, bytes_read
                                    );
                                    let chunk = UploadVideoRequest {
                                        data: Some(Data::Chunk(VideoChunk {
                                            chunk_number,
                                            data: buffer[..bytes_read].to_vec(),
                                            file_name: video_id.clone(),
                                        })),
                                    };

                                    if let Err(e) = tx.send(chunk).await {
                                        eprintln!(
                                            "❌ Failed to send chunk {} of {}: {}",
                                            chunk_number, file_name, e
                                        );
                                        chunk_send_success = false;
                                        break;
                                    }

                                    println!("➡️ Sent chunk {} of {}", chunk_number, file_name);
                                    chunk_number += 1;
                                }

                                drop(reader);

                                drop(tx);

                                if !chunk_send_success {
                                    None
                                } else {
                                    println!(
                                        "📡 Uploading {} via gRPC... (total chunks: {})",
                                        file_name, chunk_number
                                    );
                                    Some(
                                        rpc.clone()
                                            .upload_video(Request::new(ReceiverStream::new(rx)))
                                            .await,
                                    )
                                }
                            };

                            match upload_result {
                                Some(Ok(res)) => {
                                    println!("✅ Upload complete for {}: {:?}", file_name, res);
                                    println!("✅ Upload dd for {}: {:?}", file_name, path);

                                    // match tokio::fs::remove_file(path).await {
                                    //     Ok(_) => println!(
                                    //         "📁 Video file {} deleted successfully.",
                                    //         file_name
                                    //     ),
                                    //     Err(e) => eprintln!(
                                    //         "❌ Failed to delete video {}: {}",
                                    //         file_name, e
                                    //     ),
                                    // }
                                }
                                Some(Err(e)) => {
                                    eprintln!("❌ Upload failed for {}: {}", file_name, e);
                                }
                                None => {
                                    eprintln!(
                                        "❌ Chunk sending failed for {}, skipping upload",
                                        file_name
                                    );
                                }
                            }

                            // Add a small delay to prevent overwhelming the system
                            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                        }

                        let _ = rmq
                            .send_message("verify_nsfw", to_nsfw_path[0].as_bytes())
                            .await;

                        map.remove(&video_id);

                        match tokio::fs::remove_dir_all(&video_id).await {
                            Ok(_) => {
                                println!("📁 Directory {:?} deleted successfully.", &video_id);
                            }
                            Err(e) => {
                                eprintln!("❌ Failed to delete directory {:?}: {}", &video_id, e)
                            }
                        }
                        match tokio::fs::remove_file(format!("{}_master.m3u8", &video_id)).await {
                            Ok(_) => println!("📁 Video file {} deleted successfully.", &video_id),
                            Err(e) => eprintln!("❌ Failed to delete video {}: {}", &video_id, e),
                        }
                        match tokio::fs::remove_file(format!("{}_360p.mp4", &video_id)).await {
                            Ok(_) => println!("📁 Video file {} deleted successfully.", &video_id),
                            Err(e) => eprintln!("❌ Failed to delete video {}: {}", &video_id, e),
                        }
                        match tokio::fs::remove_file(format!("{}.mp4", &video_id)).await {
                            Ok(_) => println!("📁 Video file {} deleted successfully.", &video_id),
                            Err(e) => eprintln!("❌ Failed to delete video {}: {}", &video_id, e),
                        }
                    }
                }
            }
            Err(e) => eprintln!("Kafka error: {}", e),
        }
    }
}

use crate::consts::RESOLUTIONS;
use crate::video::{generate_resolutions, save_video};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::Headers;
use rdkafka::message::Message;
use rdkafka::util::get_rdkafka_version;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio_stream::StreamExt;

type ChunkIndex = usize;
type TotalChunks = usize;

pub async fn consume_video_chunks() {
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
                        let file_paths = generate_resolutions(
                            format!("{}.mp4", &video_id).as_str(),
                            &video_id,
                            RESOLUTIONS,
                        );

                        map.remove(&video_id);
                        println!("Generated resolutions for '{}': {:?}", video_id, file_paths);
                    }
                }
            }
            Err(e) => eprintln!("Kafka error: {}", e),
        }
    }
}

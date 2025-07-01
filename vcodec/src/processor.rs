use crate::consts::{NSFW_RESOLUTIONS, RESOLUTIONS};
use crate::repo::{upload_video_request::Data, UploadVideoRequest, VideoChunk, VideoMetadata};
use crate::rmq::RabbitMQ;
use crate::video::save_video;
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

// Helper function to upload a single file
async fn upload_file(
    rpc_client: Arc<crate::rpc::RpcClient>,
    file_path: &str,
    video_id: &str,
    title: &str,
    user_id: &str,
    description: &str,
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let file_size = tokio::fs::metadata(file_path)
        .await
        .map(|m| m.len())
        .unwrap_or(0);

    let (tx, rx) = mpsc::channel(16);
    let rpc = rpc_client.get_client();

    let upload_handle = tokio::spawn({
        let mut rpc = rpc.clone();
        let stream = ReceiverStream::new(rx);
        async move { rpc.upload_video(Request::new(stream)).await }
    });

    let metadata = UploadVideoRequest {
        data: Some(Data::Metadata(VideoMetadata {
            user_id: user_id.to_string(),
            title: title.to_string(),
            description: description.to_string(),
            file_name: file_path.to_string(),
            file_size: file_size as i64,
        })),
    };

    tx.send(metadata)
        .await
        .map_err(|e| format!("Failed to send metadata: {}", e))?;

    let file = File::open(file_path)
        .await
        .map_err(|e| format!("Could not open file: {}", e))?;
    let mut reader = BufReader::new(file);
    let mut buffer = vec![0u8; 1024 * 1024];
    let mut chunk_number = 0;

    loop {
        let bytes_read = match reader.read(&mut buffer).await {
            Ok(0) => break,
            Ok(n) => n,
            Err(e) => return Err(format!("Failed to read chunk: {}", e).into()),
        };

        println!(
            "📦 Reading chunk {} from {} ({} bytes)",
            chunk_number, file_path, bytes_read
        );

        let chunk = UploadVideoRequest {
            data: Some(Data::Chunk(VideoChunk {
                chunk_number,
                data: buffer[..bytes_read].to_vec(),
                file_name: video_id.to_string(),
            })),
        };

        tx.send(chunk)
            .await
            .map_err(|e| format!("Failed to send chunk: {}", e))?;
        println!("➡️ Sent chunk {} of {}", chunk_number, file_path);
        chunk_number += 1;
    }

    drop(tx);

    match upload_handle.await {
        Ok(Ok(res)) => {
            println!("✅ Upload complete for {}: {:?}", file_path, res);
            Ok(())
        }
        Ok(Err(e)) => Err(format!("Upload failed: {}", e).into()),
        Err(e) => Err(format!("Upload task panicked: {}", e).into()),
    }
}

// Modified generate_resolutions that uploads as it generates
async fn generate_and_upload_resolutions(
    input_file: &str,
    filename: &str,
    resolutions: &'static [(u32, u8)],
    rpc_client: Arc<crate::rpc::RpcClient>,
    video_id: &str,
    title: &str,
    user_id: &str,
    description: &str,
    rmq: &RabbitMQ,
) -> Vec<String> {
    let paths = Arc::new(Mutex::new(vec![]));
    let upload_tasks = Arc::new(Mutex::new(vec![]));

    let input_file = input_file.to_string();
    let filename = filename.to_string();

    let generation_tasks: Vec<_> = resolutions
        .iter()
        .map(|(height, crf)| {
            let input_file = input_file.clone();
            let filename = filename.clone();
            let paths = Arc::clone(&paths);
            let upload_tasks = Arc::clone(&upload_tasks);
            let rpc_client = rpc_client.clone();
            let video_id = video_id.to_string();
            let title = title.to_string();
            let user_id = user_id.to_string();
            let description = description.to_string();

            async move {
                let output_file = format!("{}_{}p.mp4", filename, height);
                let scale_filter = format!("scale=-2:{}", height);

                let status = tokio::process::Command::new("ffmpeg")
                    .args([
                        "-i",
                        &input_file,
                        "-vf",
                        &scale_filter,
                        "-c:v",
                        "libx264",
                        "-crf",
                        &crf.to_string(),
                        "-preset",
                        "ultrafast",
                        "-c:a",
                        "aac",
                        "-b:a",
                        "128k",
                        "-y",
                        &output_file,
                    ])
                    .status()
                    .await;

                match status {
                    Ok(status) if status.success() => {
                        println!("✅ Created {}", output_file);
                        paths.lock().unwrap().push(output_file.clone());

                        // Start upload immediately after generation
                        let upload_task = tokio::spawn({
                            let rpc_client = rpc_client.clone();
                            let output_file = output_file.clone();
                            let video_id = video_id.clone();
                            let title = title.clone();
                            let user_id = user_id.clone();
                            let description = description.clone();

                            async move {
                                println!("📦 Uploading resolution file: {}", output_file);
                                if let Err(e) = upload_file(
                                    rpc_client.clone(),
                                    &output_file,
                                    &video_id,
                                    &title,
                                    &user_id,
                                    &description,
                                )
                                .await
                                {
                                    eprintln!("❌ Upload failed for {}: {}", output_file, e);
                                }
                            }
                        });

                        upload_tasks.lock().unwrap().push(upload_task);
                    }
                    Ok(_) | Err(_) => {
                        eprintln!("❌ Failed for {}p", height);
                    }
                }
            }
        })
        .collect();

    // Wait for all generation tasks
    futures::future::join_all(generation_tasks).await;

    // Wait for all upload tasks to complete
    let tasks = upload_tasks.lock().unwrap().drain(..).collect::<Vec<_>>();
    futures::future::join_all(tasks).await;

    // Add the original file and upload it
    let mut final_paths = paths.lock().unwrap().clone();
    final_paths.push(input_file.clone());

    // Upload original file
    println!("📦 Uploading original file: {}", input_file);
    if let Err(e) = upload_file(
        rpc_client,
        &input_file,
        video_id,
        title,
        user_id,
        description,
    )
    .await
    {
        eprintln!("❌ Upload failed for original file: {}", e);
    }

    println!("📁 Generated and uploaded files: {:?}", final_paths);
    final_paths
}

// Modified generate_segments that uploads as it generates
async fn generate_and_upload_segments(
    input_file: &str,
    filename: &str,
    resolutions: &'static [(u32, u8)],
    rpc_client: Arc<crate::rpc::RpcClient>,
    video_id: &str,
    title: &str,
    user_id: &str,
    description: &str,
    rmq: Arc<RabbitMQ>, // Changed from &RabbitMQ to Arc<RabbitMQ>
) -> Vec<String> {
    let semaphore = Arc::new(tokio::sync::Semaphore::new(2));
    let mut handles = vec![];
    let upload_tasks = Arc::new(Mutex::new(vec![]));

    for (height, crf) in resolutions.iter().cloned() {
        let input_file = input_file.to_string();
        let filename = filename.to_string();
        let permit = semaphore.clone().acquire_owned().await.unwrap();
        let upload_tasks = Arc::clone(&upload_tasks);
        let rpc_client = rpc_client.clone();
        let video_id = video_id.to_string();
        let title = title.to_string();
        let user_id = user_id.to_string();
        let description = description.to_string();
        let rmq = rmq.clone(); // Clone the Arc

        handles.push(tokio::spawn(async move {
            let _permit = permit;

            let output_dir = format!("{}/{}/", filename, height);
            let playlist_file = format!("{}index.m3u8", output_dir);
            let scale_filter = format!("scale=-2:{}", height);
            let bandwidth = match height {
                144 => 200_000,
                240 => 400_000,
                360 => 800_000,
                480 => 1_000_000,
                720 => 1_500_000,
                1080 => 3_000_000,
                _ => 800_000,
            };

            std::fs::create_dir_all(&output_dir).ok();

            let status = tokio::process::Command::new("ffmpeg")
                .args([
                    "-i",
                    &input_file,
                    "-vf",
                    &scale_filter,
                    "-c:v",
                    "libx264",
                    "-crf",
                    &crf.to_string(),
                    "-preset",
                    "ultrafast",
                    "-c:a",
                    "aac",
                    "-b:a",
                    "128k",
                    "-f",
                    "hls",
                    "-hls_time",
                    "4",
                    "-hls_playlist_type",
                    "vod",
                    "-hls_segment_filename",
                    &format!("{}/seg_%03d.ts", output_dir),
                    &playlist_file,
                ])
                .status()
                .await;

            if status.map(|s| s.success()).unwrap_or(false) {
                println!("✅ Created HLS playlist: {}", playlist_file);

                let mut paths = vec![playlist_file.clone()];
                if let Ok(entries) = std::fs::read_dir(&output_dir) {
                    for entry in entries.flatten() {
                        let path = entry.path();
                        if path.extension().and_then(|s| s.to_str()) == Some("ts") {
                            if let Some(path_str) = path.to_str() {
                                paths.push(path_str.to_string());
                            }
                        }
                    }
                }

                // Upload all generated files immediately
                for (index, path) in paths.iter().enumerate() {
                    let upload_task = tokio::spawn({
                        let rpc_client = rpc_client.clone();
                        let path = path.clone();
                        let video_id = video_id.clone();
                        let title = title.clone();
                        let user_id = user_id.clone();
                        let description = description.clone();
                        let paths_clone = paths.clone();
                        let rmq = rmq.clone(); // Clone the Arc for this task

                        async move {
                            println!("📦 Uploading segment file: {}", path);
                            if let Err(e) = upload_file(
                                rpc_client.clone(),
                                &path,
                                &video_id,
                                &title,
                                &user_id,
                                &description,
                            )
                            .await
                            {
                                eprintln!("❌ Upload failed for {}: {}", path, e);
                            }

                            let progress = (index + 1) * 100 / paths_clone.len(); // Removed unnecessary parentheses
                            let _ = rmq
                                .send_progress_message(
                                    video_id.clone(),
                                    "processing".to_string(),
                                    progress as u8,
                                    user_id.clone(),
                                    "Receiving video chunks".to_string(), // Fixed typo: "Recieving" -> "Receiving"
                                    "video_processor".to_string(),
                                )
                                .await;
                        }
                    });

                    upload_tasks.lock().unwrap().push(upload_task);
                }

                let entry = format!(
                    "#EXT-X-STREAM-INF:BANDWIDTH={},RESOLUTION=1280x{}\n{}/{}/index.m3u8",
                    bandwidth, height, filename, height
                );

                Some((paths, entry))
            } else {
                eprintln!("❌ Failed for {}p", height);
                None
            }
        }));
    }

    let mut all_paths = vec![];
    let mut master_entries = vec![];

    for handle in handles {
        if let Ok(Some((paths, entry))) = handle.await {
            all_paths.extend(paths);
            master_entries.push(entry);
        }
    }

    // Write and upload master playlist
    let master_path = format!("{}_master.m3u8", filename);
    let mut master_playlist = String::from("#EXTM3U\n");
    for entry in &master_entries {
        master_playlist.push_str(entry);
        master_playlist.push('\n');
    }
    tokio::fs::write(&master_path, master_playlist)
        .await
        .expect("Failed to write master playlist");
    all_paths.insert(0, master_path.clone());

    // Upload master playlist
    let upload_task = tokio::spawn({
        let rpc_client = rpc_client.clone();
        let master_path = master_path.clone();
        let video_id = video_id.to_string();
        let title = title.to_string();
        let user_id = user_id.to_string();
        let description = description.to_string();

        async move {
            println!("📦 Uploading master playlist: {}", master_path);
            if let Err(e) = upload_file(
                rpc_client.clone(),
                &master_path,
                &video_id,
                &title,
                &user_id,
                &description,
            )
            .await
            {
                eprintln!("❌ Upload failed for master playlist: {}", e);
            }
        }
    });

    upload_tasks.lock().unwrap().push(upload_task);

    // Wait for all upload tasks to complete
    let tasks = upload_tasks.lock().unwrap().drain(..).collect::<Vec<_>>();
    futures::future::join_all(tasks).await;

    println!("📜 Master playlist: {}", master_path);
    println!("📜 All paths: {:?}", all_paths);
    all_paths
}

pub async fn consume_video_chunks(
    rpc_client: Arc<crate::rpc::RpcClient>,
    rmq: crate::rmq::RabbitMQ,
) {
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

    let video_map: Arc<Mutex<HashMap<String, (HashMap<usize, Vec<u8>>, usize)>>> =
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

                let mut chunk_index = None;
                let mut total_chunks = None;
                let mut title = None;
                let mut user_id = None;
                let mut description = None;

                if let Some(headers) = msg.headers() {
                    for i in 0..headers.count() {
                        let header = headers.get(i);
                        match header.key {
                            "chunk_index" => {
                                chunk_index = header
                                    .value
                                    .and_then(|val| String::from_utf8_lossy(val).parse().ok());
                            }
                            "total_chunks" => {
                                total_chunks = header
                                    .value
                                    .and_then(|val| String::from_utf8_lossy(val).parse().ok());
                            }
                            "title" => {
                                title = header
                                    .value
                                    .map(|val| String::from_utf8_lossy(val).to_string());
                            }
                            "user_id" => {
                                user_id = header
                                    .value
                                    .map(|val| String::from_utf8_lossy(val).to_string());
                            }
                            "description" => {
                                description = header
                                    .value
                                    .map(|val| String::from_utf8_lossy(val).to_string());
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
                        println!(
                            "✅ Received all chunks for video '{}'. Processing...",
                            video_id
                        );
                        let _ = rmq
                            .clone()
                            .send_progress_message(
                                video_id.clone(),
                                "processing".to_string(),
                                10,
                                user_id.clone().unwrap_or("unknown".to_string()),
                                "Recieving video chunks".to_string(),
                                "video_processor".to_string(),
                            )
                            .await;
                        save_video(&video_id, &entry.0).await;

                        let title_ = title.clone().unwrap_or("Untitled".to_string());
                        let user_id_ = user_id.clone().unwrap_or("unknown".to_string());
                        let description_ = description.clone().unwrap_or_default();

                        // Spawn both tasks concurrently
                        let resolution_task = tokio::spawn({
                            let rpc_client = rpc_client.clone();
                            let video_id = video_id.clone();
                            let title = title_.clone();
                            let user_id = user_id_.clone();
                            let description = description_.clone();
                            let rmq_clone = rmq.clone(); // Clone rmq here
                            let user_id_ = user_id_.clone();

                            async move {
                                println!("🎬 Starting resolution generation and upload...");
                                let nsfw_resolution_paths = generate_and_upload_resolutions(
                                    &format!("{}.mp4", &video_id),
                                    &video_id,
                                    RESOLUTIONS,
                                    rpc_client.clone(),
                                    &video_id,
                                    &title,
                                    &user_id,
                                    &description,
                                    &rmq_clone, // Use the cloned rmq
                                )
                                .await;
                                println!("✅ Resolution generation and upload complete");
                                let _ = rmq_clone
                                    .send_progress_message(
                                        video_id.clone(),
                                        "processing".to_string(),
                                        30,
                                        user_id_.clone(),
                                        "Finished generating pre check resolution".to_string(),
                                        "video_processor".to_string(),
                                    )
                                    .await;
                                nsfw_resolution_paths
                            }
                        });

                        let segment_task = tokio::spawn({
                            let rpc_client = rpc_client.clone();
                            let video_id = video_id.clone();
                            let title = title_.clone();
                            let user_id = user_id_.clone();
                            let description = description_.clone();
                            let rmq_clone = rmq.clone(); // Clone rmq here

                            async move {
                                println!("🎬 Starting segment generation and upload...");
                                let _segment_paths = generate_and_upload_segments(
                                    &format!("{}.mp4", &video_id),
                                    &video_id,
                                    NSFW_RESOLUTIONS,
                                    rpc_client.clone(),
                                    &video_id,
                                    &title,
                                    &user_id,
                                    &description,
                                    Arc::new(rmq_clone), // Use the cloned rmq
                                )
                                .await;

                                println!("✅ Segment generation and upload complete");
                            }
                        });

                        // Wait for both tasks to complete
                        let (resolution_result, _) = tokio::join!(resolution_task, segment_task);

                        // Send NSFW verification message
                        if let Ok(nsfw_resolution_paths) = resolution_result {
                            if !nsfw_resolution_paths.is_empty() {
                                let _ = rmq
                                    .clone()
                                    .send_message(
                                        "verify_nsfw",
                                        nsfw_resolution_paths[0].as_bytes(),
                                    )
                                    .await;
                                println!("Message sent to queue `verify_nsfw`");
                            }
                        }

                        // Cleanup
                        map.remove(&video_id);

                        if let Err(e) = tokio::fs::remove_dir_all(&video_id).await {
                            eprintln!("❌ Failed to delete directory {:?}: {}", &video_id, e)
                        } else {
                            println!("📁 Directory {:?} deleted successfully.", &video_id);
                        }

                        for file_path in [
                            format!("{}_master.m3u8", &video_id),
                            format!("{}_360p.mp4", &video_id),
                            format!("{}.mp4", &video_id),
                        ] {
                            match tokio::fs::remove_file(&file_path).await {
                                Ok(_) => {
                                    println!("📁 Video file {} deleted successfully.", file_path)
                                }
                                Err(e) => {
                                    eprintln!("❌ Failed to delete video {}: {}", file_path, e)
                                }
                            }
                        }
                    }
                }
            }
            Err(e) => eprintln!("Kafka error: {}", e),
        }
    }
}

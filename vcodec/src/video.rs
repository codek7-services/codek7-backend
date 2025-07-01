use std::collections::HashMap;
use std::fs::File;
use std::io::Write;

type ChunkIndex = usize;

pub async fn save_video(video_id: &str, chunks: &HashMap<ChunkIndex, Vec<u8>>) {
    let path = format!("{}.mp4", video_id);
    let mut file = File::create(&path).expect("Failed to create video file");

    for index in 0..chunks.len() {
        if let Some(chunk) = chunks.get(&index) {
            file.write_all(chunk).expect("Failed to write chunk");
        } else {
            eprintln!("❌ Missing chunk {} for video {}", index, video_id);
            return;
        }
    }

    println!("🎥 Video '{}' saved to '{}'", video_id, path);
}

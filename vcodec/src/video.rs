use rayon::prelude::*;
use std::collections::HashMap;
use std::fs::File;
use std::io::Write;
use std::process::Command;

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
/// Generate downscaled videos at specified resolutions.
/// Each resolution is a tuple: (target_height, crf_quality)
pub fn generate_resolutions(input_file: &str, filename: &str, resolutions: &'static [(u32, u8)]) -> Vec<String> {
    let mut paths: Vec<String> = resolutions
        .par_iter()
        .filter_map(|(height, crf)| {
            let output_file = format!("{}_{}p.mp4", filename, height);
            let scale_filter = format!("scale=-2:{}", height);

            let status = Command::new("ffmpeg")
                .args([
                    "-i",
                    input_file,
                    "-vf",
                    &scale_filter,
                    "-c:v",
                    "libx264",
                    "-crf",
                    &crf.to_string(),
                    "-preset",
                    "fast",
                    "-c:a",
                    "aac",
                    "-b:a",
                    "128k",
                    "-y",
                    &output_file,
                ])
                .status()
                .expect("Failed to execute ffmpeg");

            if status.success() {
                println!("✅ Created {}", output_file);
                Some(output_file)
            } else {
                eprintln!("❌ Failed for {}p", height);
                None
            }
        })
        .collect();

    paths.push(input_file.to_string());
    println!("📁 Generated files: {:?}", paths);
    paths
}



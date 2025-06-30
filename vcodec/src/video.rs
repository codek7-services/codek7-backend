use rayon::prelude::*;
use std::collections::HashMap;
use std::fs::write;
use std::fs::File;
use std::fs::{self};
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
pub fn generate_segments(
    input_file: &str,
    filename: &str,
    resolutions: &'static [(u32, u8)],
) -> Vec<String> {
    let results: Vec<_> = resolutions
        .par_iter()
        .filter_map(|(height, crf)| {
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

            fs::create_dir_all(&output_dir).unwrap();

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
                    "veryfast",
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
                .expect("Failed to execute ffmpeg");

            if status.success() {
                println!("✅ Created HLS playlist: {}", playlist_file);

                let mut paths = vec![playlist_file.clone()];
                if let Ok(entries) = fs::read_dir(&output_dir) {
                    for entry in entries.flatten() {
                        let path = entry.path();
                        if path.extension().and_then(|s| s.to_str()) == Some("ts") {
                            if let Some(path_str) = path.to_str() {
                                paths.push(path_str.to_string());
                            }
                        }
                    }
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
        })
        .collect();

    // Separate paths and master playlist entries
    let (paths_lists, master_entries): (Vec<Vec<String>>, Vec<String>) = results.into_iter().unzip();

    // Create master.m3u8
    let mut master_playlist = String::from("#EXTM3U\n");
    for entry in &master_entries {
        master_playlist.push_str(entry);
        master_playlist.push('\n');
    }

    let master_path = format!("{}_master.m3u8", filename);
    write(&master_path, master_playlist).expect("Failed to write master playlist");
    println!("📜 Master playlist: {}", master_path);

    // Flatten all paths and include master playlist path
    let mut all_paths = vec![master_path];
    for mut paths in paths_lists {
        all_paths.append(&mut paths);
    }

    println!("📜 Paths: {:?}", all_paths);
    all_paths
}

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

    println!("📁 Generated files: {:?}", paths);
    paths
}

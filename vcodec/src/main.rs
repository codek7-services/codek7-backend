use rayon::prelude::*;
use std::process::Command;

/// Generate downscaled videos at specified resolutions.
/// Each resolution is a tuple: (target_height, crf_quality)
fn generate_resolutions(input_file: &str, resolutions: &[(u32, u8)]) {
    resolutions.par_iter().for_each(|(height, crf)| {
        let output_file = format!("output_{}p.mp4", height);
        let scale_filter = format!("scale=-2:{}", height); 

        let status = Command::new("ffmpeg")
            .args([
                "-i", input_file,
                "-vf", &scale_filter,
                "-c:v", "libx264",
                "-crf", &crf.to_string(),
                "-preset", "fast",
                "-c:a", "aac",
                "-b:a", "128k",
                "-y",
                &output_file,
            ])
            .status()
            .expect("Failed to execute ffmpeg");

        if status.success() {
            println!("✅ Created {}", output_file);
        } else {
            eprintln!("❌ Failed for {}p", height);
        }
    });
}

fn main() {
    let input_file = "video.mp4";

    // Customizable list of (height, CRF quality)
    let resolutions = vec![
        (144, 35),
        (240, 32),
        (360, 29),
        (480, 26),
        (720, 23),
        (1080, 20),
    ];

    generate_resolutions(input_file, &resolutions);
}


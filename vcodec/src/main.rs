mod kafka;
mod video;
mod consts;
use crate::kafka::consume_video_chunks;

#[tokio::main]
async fn main() {
    consume_video_chunks().await;
}


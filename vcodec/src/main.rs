pub mod repo {
    tonic::include_proto!("repo"); 
}

mod consts;
mod kafka;
mod rpc;
mod video;
use crate::kafka::consume_video_chunks;

#[tokio::main]
async fn main() {
    let rpc_client = rpc::RpcClient::new().await.expect("Failed to create RPC client");
    consume_video_chunks(rpc_client).await;
}

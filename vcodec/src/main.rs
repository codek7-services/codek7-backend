pub mod repo {
    tonic::include_proto!("repo");
}

mod consts;
mod kafka;
mod rmq;
mod rpc;
mod video;
use crate::kafka::consume_video_chunks;
use crate::rmq::RabbitMQ;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let rpc_client = rpc::RpcClient::new()
        .await
        .expect("Failed to create RPC client");
    let uri = "amqp://guest:guest@127.0.0.1:5672/%2f";
    let queue = "verify_nsfw";

    let rmq = RabbitMQ::connect(uri, queue).await?;
    consume_video_chunks(rpc_client, rmq).await;

    Ok(())
}

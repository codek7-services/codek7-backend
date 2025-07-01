pub mod repo {
    tonic::include_proto!("repo");
}

mod consts;
mod processor;
mod rmq;
mod rpc;
mod video;
use std::sync::Arc;

use crate::processor::consume_video_chunks;
use crate::rmq::RabbitMQ;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let rpc_client = Arc::new(
        rpc::RpcClient::new()
            .await
            .expect("Failed to create RPC client"),
    );
    let uri = "amqp://guest:guest@127.0.0.1:5672/%2f";
    let queue = "verify_nsfw";
    let queue2 = "notify.q";

    let rmq = RabbitMQ::connect(uri, queue, queue2).await?;
    consume_video_chunks(rpc_client, rmq).await;

    Ok(())
}

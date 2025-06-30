use crate::repo::repo_service_client::RepoServiceClient;
use tonic::transport::Channel;

pub struct RpcClient {
    client: RepoServiceClient<Channel>,
}

impl RpcClient {
    pub async fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let client = RepoServiceClient::connect("http://[::1]:50051").await?;
        Ok(RpcClient { client })
    }

    pub fn get_client(&self) -> &RepoServiceClient<Channel> {
        &self.client
    }
}


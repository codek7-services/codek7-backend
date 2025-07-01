use anyhow::Result;
use lapin::{
    options::{BasicPublishOptions, QueueDeclareOptions},
    types::FieldTable,
    BasicProperties, Channel, Connection, ConnectionProperties,
};
use serde::Serialize;

#[derive(Serialize)]
struct ProgressMessage {
    video_id: String,
    event_type: String,
    progress: u8, // percent
    user_id: String,
    description: String,
    service_name: String,
}
#[derive(Debug, Clone)]
pub struct RabbitMQ {
    channel: Channel,
}

impl RabbitMQ {
    pub async fn connect(uri: &str, queue1: &str, queue2: &str) -> Result<Self> {
        let conn = Connection::connect(uri, ConnectionProperties::default()).await?;
        println!("✅ Connected to RabbitMQ");

        let channel = conn.create_channel().await?;

        // Declare first queue
        channel
            .queue_declare(
                queue1,
                QueueDeclareOptions::default(),
                FieldTable::default(),
            )
            .await?;

        channel
            .queue_declare(
                queue2,
                QueueDeclareOptions::default(),
                FieldTable::default(),
            )
            .await?;

        Ok(Self { channel })
    }

    pub async fn send_progress_message(
        &self,
        video_id: String,
        event_type: String,
        progress: u8,
        user_id: String,
        description: String,
        service_name: String,
    ) -> Result<()> {
        let message = ProgressMessage {
            video_id,
            event_type,
            progress,
            user_id,
            description,
            service_name,
        };

        let body = serde_json::to_vec(&message)?;

        self.send_message("notify.q", &body).await
    }
    pub async fn send_message(&self, queue_name: &str, body: &[u8]) -> Result<()> {
        self.channel
            .basic_publish(
                "",
                queue_name,
                BasicPublishOptions::default(),
                body,
                BasicProperties::default(),
            )
            .await?
            .await?; // wait for confirmation

        println!("Message sent to queue `{}`", queue_name);
        Ok(())
    }

    pub fn channel(&self) -> &Channel {
        &self.channel
    }
}

use lapin::{
    options::{BasicPublishOptions, QueueDeclareOptions},
    types::FieldTable,
    BasicProperties, Channel, Connection, ConnectionProperties,
};
use anyhow::Result;

pub struct RabbitMQ {
    channel: Channel,
}

impl RabbitMQ {
    pub async fn connect(uri: &str, queue_name: &str) -> Result<Self> {
        let conn = Connection::connect(uri, ConnectionProperties::default()).await?;
        println!("Connected to RabbitMQ");

        let channel = conn.create_channel().await?;

        channel
            .queue_declare(
                queue_name,
                QueueDeclareOptions::default(),
                FieldTable::default(),
            )
            .await?;

        Ok(Self { channel })
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


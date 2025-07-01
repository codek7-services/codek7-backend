<a id="readme-top"></a>
<br />

<h3 align="center">Codek7 : the Self-Hosted Media Streaming Platform</h3>

  <p align="center">
    A robust, self-hosted media streaming platform with automatic transcoding and adaptive bitrate delivery.
    <br />
    <a href="#about-the-project"><strong>Explore the features »</strong></a>
    <br />
    <a href="#getting-started">Get Started</a>
    <br/>
    <a href="https://deepwiki.com/lumbrjx/codek7/1-overview" >Wiki</a>
  </p>
</div>

---

<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#features">Features</a></li>
        <li><a href="#technologies-used">Technologies Used</a></li>
        <li><a href="#architecture-high-level">Architecture (High-Level)</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#steps-to-run">Steps to Run</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#system-diagrams">System Diagrams</a></li>
    <li><a href="#data-model">Data Model</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
  </ol>
</details>

---

## About The Project

This project delivers a powerful, self-hosted media streaming solution, designed for seamless content delivery. It intelligently handles **automatic transcoding** of various media formats and provides **adaptive bitrate delivery** (HLS), ensuring a smooth and high-quality viewing experience across diverse devices and network conditions.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Features

* **Automatic Transcoding:** Media files are automatically transcoded into multiple formats and resolutions, optimizing them for streaming.
* **Adaptive Bitrate (ABR) Delivery:** Utilizes HLS (HTTP Live Streaming) to dynamically adjust video quality based on the user's network speed, minimizing buffering and maximizing viewing pleasure.
* **Scalable Architecture:** Built with a microservices approach, leveraging message queues for efficient processing and scalability.
* **Object Storage Integration:** Stores transcoded media files in a high-performance, S3-compatible object storage solution.
* **Modern User Interface:** A responsive and intuitive web interface for managing and Browse your media library.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Technologies Used

* [![Go][Golang]][Golang-url]
* [![Rust][Rust]][Rust-url]
* [![Kafka][Kafka]][Kafka-url]
* [![RabbitMQ][RabbitMQ]][RabbitMQ-url]
* [![FFmpeg][FFmpeg]][FFmpeg-url]
* [![HLS][HLS]][HLS-url]
* [![Next.js][Nextjs]][Nextjs-url]
* [![MinIO][MinIO]][MinIO-url]
* [![PostgreSQL][PostgreSQL]][PostgreSQL-url]
* [![Redis][Redis]][Redis-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Architecture (High-Level)

The platform follows a distributed microservices architecture:

1.  **Ingestion Service (Go/Rust):** Handles uploading and initial processing of media files. Publishes events to Kafka.
2.  **Transcoding Service (Go/Rust with FFmpeg):** Consumes events from Kafka, performs transcoding using FFmpeg, and stores transcoded segments in MinIO. Publishes completion events to Kafka.
3.  **API Gateway/Streaming Service (Go):** Provides APIs for the frontend and serves HLS manifests and segments directly from MinIO.
4.  **Frontend (Next.js):** Communicates with the API Gateway to display media, manage uploads, and initiate playback.
5.  **Database Layer (PostgreSQL & Redis):** PostgreSQL stores persistent data, while Redis handles caching and real-time data.
6.  **Message Queues (Kafka & RabbitMQ):** Facilitate communication and task distribution between services, ensuring high availability and scalability.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Getting Started

To get a local copy of this streaming platform up and running, follow these simple steps.

### Prerequisites

* Docker and Docker Compose (essential for running all services)

### Steps to Run

1.  **Clone the project repository:**
    ```bash
    cd codek7-backend
    ```
2. **Build the vcodec service:
   ```bash
    cd vcodec
   cargo build
    ```
2.  **Start all services using Docker Compose:**
    ```bash
    docker compose up
    ```
    This command will build (if necessary) and start all the services defined in  `docker-compose.yml` file, including Kafka, RabbitMQ, PostgreSQL, Redis, MinIO, your custom Go/Rust backend services, and the Next.js frontend application.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Usage

* Once all services are up and running, you can access the frontend application in your web browser. Typically, it will be available at `http://localhost:3000`, but check your `docker-compose.yml` or frontend configuration for the exact port.
* Upload your media files through the user interface.
* The platform will automatically transcode and prepare your media for adaptive bitrate streaming.
* Browse your media library and enjoy a smooth viewing experience.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## System Diagrams

Below are various diagrams illustrating the system's architecture, data flow, and video processing states.

### Overall System Architecture

This diagram provides a high-level overview of the main components and their interactions.

<div align="center">
  <img src="https://media.discordapp.net/attachments/1385955155173838860/1389715381643513956/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text._A_smarter_way_of_creating_diagrams.-2025-07-01-211236.png?ex=6865a0ce&is=68644f4e&hm=ae55d9526ea941835b926ab030f6434cba730c26f2875c47114365a4fb6f6fe2&=&format=webp&quality=lossless&width=2844&height=1048" alt="Overall System Architecture">
  <br/>
  <img src="https://media.discordapp.net/attachments/1385955155173838860/1389716221326528682/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text._A_smarter_way_of_creating_diagrams.-2025-07-01-211508.png?ex=6865a196&is=68645016&hm=bd9d57d8e2ce4ccda18ee7b4e72711f149c695426d7effca286442a75acf7966&=&format=webp&quality=lossless&width=1298&height=1540" alt="Simplified Architecture">
</div>
<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Video Processing Flow

This diagram details the sequence of operations from video upload to final availability, including transcoding and NSFW checking.

<div align="center">
  <img src="https://media.discordapp.net/attachments/1385955155173838860/1389716229375393923/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text._A_smarter_way_of_creating_diagrams.-2025-07-01-211548.png?ex=6865a198&is=68645018&hm=2ce32c550ab4ec05823bb9c9b547a0c2ad9eb75645e5738dbf3b92c247867be6&=&format=webp&quality=lossless&width=834&height=1538" alt="Video Processing Flow Detailed">
</div>
<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Service Interaction Sequence

This sequence diagram illustrates the communication flow between different services during a video upload and processing cycle.

<div align="center">
  <img src="https://media.discordapp.net/attachments/1385955155173838860/1389476633483673620/Editor___Mermaid_Chart-2025-07-01-052321.png?ex=68656b34&is=686419b4&hm=84f4f8f11508a6a9a0ea6f184bb7a4f0789b874c76d66b4548979b3bf615c867&=&format=webp&quality=lossless&width=1140&height=1540" alt="Service Interaction Sequence Diagram">
</div>
<p align="right">(<a href="#readme-top">back to top</a>)</p>


---

## Data Model

This entity-relationship diagram (ERD) showcases the database schema and relationships between different entities in the system.

<div align="center">
  <img src="https://media.discordapp.net/attachments/1385955155173838860/1389720763224948786/Mermaid_Chart_-_Create_complex_visual_diagrams_with_text._A_smarter_way_of_creating_diagrams.-2025-07-01-213355.png?ex=6865a5d1&is=68645451&hm=80d261fb7c9ce967efc10ba58eacd1eb6f7ae286c9069aead8f8ca05b22a07c4&=&format=webp&quality=lossless&width=1458&height=1540" alt="Database Diagram">
</div>
<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Roadmap

* [ ] User authentication and authorization
* [ ] Advanced media library management (categories, tags)
* [ ] Search and filtering capabilities
* [ ] Playback history and watch progress tracking
* [ ] API for external applications
* [ ] Multi-user support with custom profiles
* [ ] Live streaming capabilities

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## License

Distributed under the MIT License. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

[Golang]: https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white
[Golang-url]: https://go.dev/
[Rust]: https://img.shields.io/badge/rust-%23000000.svg?style=for-the-badge&logo=rust&logoColor=white
[Rust-url]: https://www.rust-lang.org/
[Kafka]: https://img.shields.io/badge/kafka-%23231F20.svg?style=for-the-badge&logo=apache-kafka&logoColor=white
[Kafka-url]: https://kafka.apache.org/
[RabbitMQ]: https://img.shields.io/badge/RabbitMQ-%23FF6600.svg?style=for-the-badge&logo=rabbitmq&logoColor=white
[RabbitMQ-url]: https://www.rabbitmq.com/
[FFmpeg]: https://img.shields.io/badge/ffmpeg-3982CE?style=for-the-badge&logo=ffmpeg&logoColor=white
[FFmpeg-url]: https://ffmpeg.org/
[HLS]: https://img.shields.io/badge/HLS-adaptive%20bitrate-F8F8F8?style=for-the-badge&logo=apple&logoColor=black
[HLS-url]: https://developer.apple.com/streaming/
[Nextjs]: https://img.shields.io/badge/next.js-000000?style=for-the-badge&logo=nextdotjs&logoColor=white
[Nextjs-url]: https://nextjs.org/
[MinIO]: https://img.shields.io/badge/minio-%23FF7D00.svg?style=for-the-badge&logo=minio&logoColor=white
[MinIO-url]: https://min.io/
[PostgreSQL]: https://img.shields.io/badge/PostgreSQL-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white
[PostgreSQL-url]: https://www.postgresql.org/
[Redis]: https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white
[Redis-url]: https://redis.io/

mod settings;

use anyhow::Result;
use clap::{Parser, Subcommand};
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;
use tokio_tungstenite::connect_async;
use tokio_tungstenite::tungstenite::protocol::Message;

#[derive(Parser)]
#[command(name = "bie")]
#[command(bin_name = "bie")]
struct Cli {
    /// Subcommand to execute
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Download a file
    Get {
        /// The name of the file to download
        file_name: PathBuf,
    },

    /// Show configuration
    Config,
}



#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse command line arguments
    let cli = Cli::parse();

    match cli.command {
        Commands::Config => {
            let settings = settings::Settings::load()?;
            println!("{:?}", settings);
        }
        Commands::Get { file_name } => {
            // // Generate the link for curl
            // let url = format!("http://your-server/upload/{}", file_name);
            // println!("Downloading file from: {}", url);

            // // Use curl to download the file
            // let status = std::process::Command::new("curl")
            //     .arg("-o")
            //     .arg(&file_name)
            //     .arg(&url)
            //     .status()?;

            // if !status.success() {
            //     eprintln!("Failed to download file.");
            //     return Ok(());
            // }

            // // Connect to the WebSocket server
            // let (ws_stream, _) = connect_async("ws://your-server/wait_file").await?;
            // println!("WebSocket connected!");

            // let (mut write, mut read) = ws_stream.split();

            // // Send a token request (modify as needed)
            // write.send(Message::Text("token_request".into())).await?;

            // // Open a file to write the received data
            // let mut output_file = File::create(&file_name)?;

            // // Receive data from WebSocket
            // while let Some(msg) = read.next().await {
            //     match msg? {
            //         Message::Binary(data) => {
            //             output_file.write_all(&data)?;
            //             println!("Received {} bytes of data.", data.len());
            //         }
            //         Message::Close(_) => {
            //             println!("WebSocket connection closed.");
            //             break;
            //         }
            //         _ => {}
            //     }
            // }

            // println!("File download complete.");
        }
    }

    Ok(())
}

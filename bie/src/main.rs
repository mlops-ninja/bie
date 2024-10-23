mod settings;

use anyhow::Result;
use clap::{Parser, Subcommand};
use std::io::Write;
use std::path::PathBuf;
use tempfile::NamedTempFile;
use tokio_tungstenite::connect_async;
use tokio_tungstenite::tungstenite::protocol::Message;
use url::Url;

use futures::{SinkExt, StreamExt};

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
async fn main() -> Result<(), anyhow::Error> {
    // Parse command line arguments
    let cli = Cli::parse();

    match cli.command {
        Commands::Config => {
            let settings = settings::Settings::load()?;
            println!("{:?}", settings);
            Ok(())
        }
        Commands::Get { file_name } => {
            let settings = settings::Settings::load()?;

            // First of all - we need to connect to the server via websocket and request a token
            let url = create_websocket_url(&settings.bastion_server_url)?;
            let ws_url = url.to_string();
            let (ws_stream, _) = connect_async(&ws_url).await?;

            let (mut write, mut read) = ws_stream.split();

            // The first message from the server is a token
            let token_message = read.next().await;
            let token_message =
                token_message.ok_or_else(|| anyhow::anyhow!("No token message received"))?;
            let token_message = token_message?;

            let token = match token_message {
                Message::Text(token) => token,
                _ => {
                    return Err(anyhow::anyhow!("Invalid token message"));
                }
            };

            // Generate the link for curl
            let url = format!("{}/upload/{}", settings.bastion_server_url, token);
            println!("In order to send a file, use this snippet:\n");
            println!("echo <your-file-name> | xargs -I{{}} curl -v -X POST -F \"file=@{{}}\" {}\n", url);
            println!("<type echo then <tab> to get file name autocompletion, then copy snippet from \"|\", or Ctrl-C to exit>");

            // Here we start a loop to write all incoming content into a file

            // First - create temp file in tmp directory
            let mut temp_file = NamedTempFile::new()?;

            // Now - start receiving loop for websocket messages
            loop {
                let message = read.next().await;
                let message = message.ok_or_else(|| anyhow::anyhow!("No message received"))?;
                let message = message?;

                match message {
                    Message::Binary(data) => {
                        // Here we should parse CBOR encoded message
                        match bie_common::BieProtocol::from(&data[..]) {
                            bie_common::BieProtocol::FileChunk(chunk) => {
                                temp_file.write_all(&chunk)?;
                            }
                            bie_common::BieProtocol::EndOfFile => {
                                break;
                            }
                            _ => {
                                return Err(anyhow::anyhow!("Invalid message"));
                            }
                        }
                    }
                    Message::Close(_) => {
                        return Ok(());
                    }
                    _ => {
                        return Err(anyhow::anyhow!("Invalid message"));
                    }
                }
            }
            // Here we need to flush the tempfile and copy it to right place
            temp_file.flush()?;
            temp_file.persist(file_name.as_path())?;

            write.close().await?;

            Ok(())
        }
    }
}

fn create_websocket_url(server_url: &str) -> Result<Url, anyhow::Error> {
    let mut url = Url::parse(server_url)?;
    // Here the scheme is valid
    if url.scheme() == "http" {
        url.set_scheme("ws")
            .map_err(|_| anyhow::anyhow!("Invalid scheme"))?;
    } else if url.scheme() == "https" {
        url.set_scheme("wss")
            .map_err(|_| anyhow::anyhow!("Invalid scheme"))?;
    } else {
        return Err(anyhow::anyhow!("Invalid scheme"));
    }

    url.set_path("/wait_file");
    Ok(url)
}

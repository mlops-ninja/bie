use log::{debug, error, info, trace};
mod settings;

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock};
use warp::filters::ws::Message;
use warp::ws::{WebSocket, Ws};
use warp::Buf;
use warp::Filter;

use futures::SinkExt;
use futures::StreamExt;

use bie_common::{generate_secure_random_string, BieProtocol};

type Result<T> = std::result::Result<T, warp::reject::Rejection>;

// Type alias for client token and connection sender
type Connections = Arc<
    RwLock<HashMap<String, mpsc::UnboundedSender<core::result::Result<BieProtocol, warp::Error>>>>,
>;

#[tokio::main]
async fn main() {
    let env = env_logger::Env::default()
        .filter_or("MY_LOG_LEVEL", "info")
        .write_style_or("MY_LOG_STYLE", "always");
    env_logger::init_from_env(env);

    let settings = settings::Settings::load().unwrap();

    // Initializing clients
    let connections: Connections = Arc::new(RwLock::new(HashMap::new()));

    // Health check route
    let health_route = warp::path!("health").and_then(health_handler);

    // Route to push actual file
    let upload_route = warp::path!("upload" / String)
        .and(warp::multipart::form().max_length(1_073_741_824))
        .and(with_connections(connections.clone()))
        .and_then(handle_upload);

    // WebSocket route to handle new connections
    let ws_route = warp::path("wait_file")
        .and(warp::ws())
        .and(with_connections(connections.clone()))
        .map(|ws: Ws, clients| ws.on_upgrade(move |socket| handle_connection(socket, clients)));

    // Start the server
    info!("Starting server on port: {}", settings.port);
    warp::serve(health_route.or(upload_route).or(ws_route))
        .run(([0, 0, 0, 0], settings.port))
        .await;
}

fn with_connections(
    connections: Connections,
) -> impl Filter<Extract = (Connections,), Error = std::convert::Infallible> + Clone {
    warp::any().map(move || connections.clone())
}

async fn handle_connection(ws: WebSocket, connections: Connections) {
    let (mut client_ws_sender, mut client_ws_rcv) = ws.split();
    let (client_sender, mut client_rcv) = mpsc::unbounded_channel();

    // Generate a unique token for this connection
    let token = generate_secure_random_string(32);

    info!("New connection with token: {}", token);

    // Send the token to the client
    match client_ws_sender.send(Message::text(token.clone())).await {
        Ok(_) => {
            debug!("Sent token to client: {}", token);
        }
        Err(e) => {
            error!("Error sending token to client: {}", e);
            let _ = client_ws_sender
                .close()
                .await
                .map_err(|e| error!("Error closing connection: {}", e));
            return;
        }
    }

    let debug_token = token.clone();
    tokio::task::spawn(async move {
        trace!("Starting loop for token: {}", debug_token);
        while let Some(Ok(msg)) = client_rcv.recv().await {
            trace!("Received message: {:?}", msg);
            match msg {
                BieProtocol::Token(token) => {
                    trace!("Received token: {}", token);
                    let cbor: Vec<u8> = BieProtocol::Token(token).into();
                    client_ws_sender.send(Message::binary(cbor)).await.unwrap();
                }
                BieProtocol::FileChunk(chunk) => {
                    trace!("Sending chunk of size: {}\n{:?}", chunk.len(), chunk);
                    let cbor: Vec<u8> = BieProtocol::FileChunk(chunk).into();
                    client_ws_sender.send(Message::binary(cbor)).await.unwrap();
                }
                BieProtocol::EndOfFile => {
                    trace!("Received end of file");
                    let cbor: Vec<u8> = BieProtocol::EndOfFile.into();
                    client_ws_sender.send(Message::binary(cbor)).await.unwrap();
                    client_rcv.close();
                    client_ws_sender.close().await.unwrap();
                }
            };
        }
    });

    {
        // Store the connection associated with the token
        connections
            .write()
            .await
            .insert(token.clone(), client_sender);

        // Ensuring that write lock is released with out of scope
    }

    // Wait for the connection to close
    while let Some(_) = client_ws_rcv.next().await {}

    // Remove the connection when closed
    connections.write().await.remove(&token);
}

async fn handle_upload(
    token: String,
    form: warp::multipart::FormData,
    connections: Connections,
) -> Result<impl warp::Reply> {
    info!("Received file upload request for token: {}", token);
    if let Some(client_tx) = connections.read().await.get(&token) {
        // Process the uploaded file
        form.for_each(|part| async {
            if let Ok(part) = part {
                trace!("Processing part: {:?}", part.name());
                if part.name() == "file" {
                    let mut data = part.stream();
                    while let Some(Ok(mut chunk)) = data.next().await {
                        // Send the file chunks to the WebSocket connection
                        while chunk.has_remaining() {
                            let mut buffer = vec![0; chunk.remaining()];
                            chunk.copy_to_slice(&mut buffer);
                            match client_tx.send(Ok(BieProtocol::FileChunk(buffer.clone()))) {
                                Ok(_) => {
                                    trace!("Sent chunk of size: {}", buffer.len());
                                }
                                Err(e) => {
                                    error!("Error sending chunk: {}", e);
                                    return;
                                }
                            }
                        }
                    }
                }
            }
        })
        .await;

        match client_tx.send(Ok(BieProtocol::EndOfFile)) {
            Ok(_) => {
                trace!("Sent end of file");
            }
            Err(e) => {
                error!("Error sending end of file: {}", e);
                return Ok(warp::http::status::StatusCode::INTERNAL_SERVER_ERROR);
            }


        }

        return Ok(warp::http::status::StatusCode::ACCEPTED);
    }
    Ok(warp::http::status::StatusCode::NOT_FOUND)
}

async fn health_handler() -> Result<impl warp::Reply> {
    Ok(warp::http::StatusCode::OK)
}
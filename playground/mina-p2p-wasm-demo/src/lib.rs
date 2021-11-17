// use anyhow::Result;
use libp2p::{
    core::{upgrade, ProtocolName},
    futures::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt, StreamExt},
    identity, noise,
    pnet::{PnetConfig, PreSharedKey},
    request_response::{
        ProtocolSupport, RequestResponse, RequestResponseCodec, RequestResponseConfig,
        RequestResponseEvent,
    },
    swarm::{NetworkBehaviourEventProcess, SwarmBuilder, SwarmEvent},
    wasm_ext::ExtTransport,
    NetworkBehaviour, PeerId, Transport,
};
use std::{io, time::Duration};
// use libp2p_relay::RelayConfig;
use multihash::{Blake2b256, StatefulHasher};
// use std::time::Duration;
use wasm_bindgen::prelude::*;

const RENDEZVOUS_STRING: &str =
    "/coda/0.0.1/5f704cc0c82e0ed70e873f0893d7e06f148524e3f0bdae2afb02e7819a0c24d1";
// const RELAY_SERVER_WS_ADDR: &str =
//     "/ip4/127.0.0.1/tcp/43637/ws/p2p/QmdDda64RhVC2BMHdW8y92jfcjWEH8qhzozHkbRt6gKXY2";
const MINA_PEER_ADDR: &str =
    "/ip4/95.217.106.189/tcp/8302/p2p/12D3KooWSxxCtzRLfUzoxgRYW9fTKWPUujdvStuwCPSPUN3629mb";
// "/ip4/127.0.0.1/tcp/8302/p2p/12D3KooWKK3RpV1MWAZk3FJ5xqbVPL2BMDdUEGSfwfQoUprBNZCv";

#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace= console, js_name = log)]
    fn log_string(s: String);
}

#[wasm_bindgen]
pub fn wasm_test() -> bool {
    log_string("wasm_test".into());
    true
}

#[wasm_bindgen]
pub async fn wasm_test_async() -> bool {
    log_string("wasm_test_async".into());
    true
}

#[wasm_bindgen]
pub async fn connect(addr: String) -> bool {
    connect_async(&addr).await
}

// #[tokio::main(flavor = "current_thread")]
async fn connect_async(addr: &str) -> bool {
    // env_logger::init();

    let js_promise = js_sys::Promise::resolve(&42.into());
    let js_future: wasm_bindgen_futures::JsFuture = js_promise.into();
    let js_val = js_future.await.unwrap();
    log_string(format!("js_val: {:?}", js_val));

    log_string(format!("Relay node ws address: {}", addr));
    log_string(format!("Mina node address: {}", MINA_PEER_ADDR));

    // Create a random PeerId
    let id_keys = identity::Keypair::generate_ed25519();
    let peer_id = PeerId::from(id_keys.public());
    log_string(format!("Local peer id: {:?}", peer_id));

    // Create a keypair for authenticated encryption of the transport.
    let noise_keys = noise::Keypair::<noise::X25519Spec>::new()
        .into_authentic(&id_keys)
        .expect("Signing libp2p-noise static DH keypair failed.");

    let mut hasher = Blake2b256::default();
    hasher.update(RENDEZVOUS_STRING.as_bytes());
    let hash = hasher.finalize();
    let psk = hash.as_ref();
    log_string(format!("psk: {}", hex::encode(psk)));
    let mut psk_fixed: [u8; 32] = Default::default();
    psk_fixed.copy_from_slice(&psk[0..32]);
    let psk = PreSharedKey::new(psk_fixed);
    let mut mux_config = libp2p_mplex::MplexConfig::new();
    mux_config.set_protocol(b"/coda/mplex/1.0.0");
    let transport = {
        let ws = ExtTransport::new(libp2p::wasm_ext::ffi::websocket_transport());
        ws.and_then(move |socket, _| PnetConfig::new(psk).handshake(socket))
            .upgrade(upgrade::Version::V1)
            .authenticate(noise::NoiseConfig::xx(noise_keys).into_authenticated())
            .multiplex(mux_config)
            .boxed()
    };

    let parsed_addr = addr.parse().unwrap();
    log_string(format!("Connecting to relay server via ws {} ... ", addr));
    let mut swarm = {
        let behaviour = NodeStatusBehaviour::new().await.unwrap();
        SwarmBuilder::new(transport, behaviour, peer_id).build()
    };
    match swarm.dial_addr(parsed_addr) {
        Ok(_) => {
            log_string(format!("dial ok"));
            // match dial.await {
            //     Ok(_) => {
            //         log_string("dial await ok".into());
            //         // return Ok(true);
            //         return true;
            //     }
            //     Err(e) => log_string(format!("Fail to dail 2: {}", e)),
            // };
            loop {
                match swarm.select_next_some().await {
                    SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                        log_string(format!("Connected to {}", peer_id));
                        swarm
                            .behaviour_mut()
                            .request_response
                            .send_request(&peer_id, NodeStatusRequest);
                    }
                    _ => {}
                }
            }
            return true;
        }
        Err(e) => log_string(format!("Fail to dail: {}", e)),
    }
    false
    // Ok(false)
}

#[derive(NetworkBehaviour)]
#[behaviour(event_process = true)]
// #[behaviour(out_event = "NodeStatusEvent")]
struct NodeStatusBehaviour {
    request_response: RequestResponse<NodeStatusCodec>,
}

impl NodeStatusBehaviour {
    async fn new() -> anyhow::Result<Self> {
        let mut config = RequestResponseConfig::default();
        config.set_request_timeout(Duration::from_secs(60));
        Ok(Self {
            request_response: RequestResponse::new(
                NodeStatusCodec,
                std::iter::once((NodeStatusProtocol, ProtocolSupport::Full)),
                config,
            ),
        })
    }
}

impl NetworkBehaviourEventProcess<RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>>
    for NodeStatusBehaviour
{
    fn inject_event(&mut self, event: RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>) {
        log_string(format!("RequestResponseEvent: {:?}", event));
    }
}

#[derive(Debug, Clone)]
struct NodeStatusProtocol;

impl ProtocolName for NodeStatusProtocol {
    fn protocol_name(&self) -> &[u8] {
        // b"/mina/node-status"
        b"/mytest"
    }
}

#[derive(Clone)]
struct NodeStatusCodec;

#[derive(Debug, Clone, PartialEq, Eq)]
struct NodeStatusRequest;

#[derive(Debug, Clone, PartialEq, Eq)]
struct NodeStatusResponse(String);

#[async_trait::async_trait]
impl RequestResponseCodec for NodeStatusCodec {
    type Protocol = NodeStatusProtocol;
    type Request = NodeStatusRequest;
    type Response = NodeStatusResponse;

    async fn read_request<T>(
        &mut self,
        _: &Self::Protocol,
        _io: &mut T,
    ) -> io::Result<Self::Request>
    where
        T: AsyncRead + Unpin + Send,
    {
        log_string(format!("read_request"));
        Ok(NodeStatusRequest)
    }

    async fn read_response<T>(
        &mut self,
        _: &Self::Protocol,
        io: &mut T,
    ) -> io::Result<Self::Response>
    where
        T: AsyncRead + Unpin + Send,
    {
        let mut json = String::new();
        io.read_to_string(&mut json).await?;
        log_string(format!("read_response: {}", json));
        Ok(NodeStatusResponse(json))
    }

    async fn write_request<T>(
        &mut self,
        _: &Self::Protocol,
        io: &mut T,
        _: Self::Request,
    ) -> io::Result<()>
    where
        T: AsyncWrite + Unpin + Send,
    {
        log_string(format!("write_request"));
        io.close().await?;
        Ok(())
    }

    async fn write_response<T>(
        &mut self,
        _: &Self::Protocol,
        io: &mut T,
        NodeStatusResponse(json): Self::Response,
    ) -> io::Result<()>
    where
        T: AsyncWrite + Unpin + Send,
    {
        log_string(format!("write_response: {}", json));
        io.write_all(json.as_bytes()).await?;
        io.close().await?;
        Ok(())
    }
}

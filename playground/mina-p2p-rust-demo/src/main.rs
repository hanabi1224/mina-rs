use anyhow::{bail, Result};
use futures::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt, StreamExt};
use libp2p::{
    core::{upgrade, ProtocolName},
    identity,
    mdns::{Mdns, MdnsEvent},
    noise,
    pnet::{PnetConfig, PreSharedKey},
    request_response::{
        ProtocolSupport, RequestResponse, RequestResponseCodec, RequestResponseConfig,
        RequestResponseEvent,
    },
    swarm::{NetworkBehaviourEventProcess, SwarmBuilder, SwarmEvent},
    tcp::TokioTcpConfig,
    websocket::WsConfig,
    NetworkBehaviour, PeerId, Transport,
};
use std::{io, time::Duration};
// use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};
// use libp2p_relay::RelayConfig;
use multihash::{Blake2b256, StatefulHasher};

const RENDEZVOUS_STRING: &str =
    "/coda/0.0.1/5f704cc0c82e0ed70e873f0893d7e06f148524e3f0bdae2afb02e7819a0c24d1";
// const RELAY_SERVER_WS_ADDR: &str =
//     "/ip4/127.0.0.1/tcp/43637/ws/p2p/QmdDda64RhVC2BMHdW8y92jfcjWEH8qhzozHkbRt6gKXY2";
const MINA_PEER_ADDR: &str =
    "/ip4/95.217.106.189/tcp/8302/p2p/12D3KooWSxxCtzRLfUzoxgRYW9fTKWPUujdvStuwCPSPUN3629mb";
// "/ip4/127.0.0.1/tcp/40661/p2p/12D3KooWEKFr7y5Gh4zHbNzwZt3QLjQq84czaGNaSJRHFMh7ufxP";

fn main() {
    main_async().unwrap();
}

#[tokio::main]
async fn main_async() -> Result<()> {
    env_logger::init();

    println!("Mina node address: {}", MINA_PEER_ADDR);

    // Create a random PeerId
    let id_keys = identity::Keypair::generate_ed25519();
    let peer_id = PeerId::from(id_keys.public());
    println!("Local peer id: {:?}", peer_id);

    // Create a keypair for authenticated encryption of the transport.
    let noise_keys = noise::Keypair::<noise::X25519Spec>::new()
        .into_authentic(&id_keys)
        .expect("Signing libp2p-noise static DH keypair failed.");

    let mut hasher = Blake2b256::default();
    hasher.update(RENDEZVOUS_STRING.as_bytes());
    let hash = hasher.finalize();
    let psk = hash.as_ref();
    println!("psk: {}", hex::encode(psk));
    let mut psk_fixed: [u8; 32] = Default::default();
    psk_fixed.copy_from_slice(&psk[0..32]);
    let psk = PreSharedKey::new(psk_fixed);
    let mut mux_config = libp2p_mplex::MplexConfig::new();
    mux_config.set_protocol(b"/coda/mplex/1.0.0");
    let transport = {
        let tcp = TokioTcpConfig::new().nodelay(true);
        let ws = WsConfig::new(tcp.clone());
        tcp.or_transport(ws)
            .and_then(move |socket, _| PnetConfig::new(psk).handshake(socket))
            .upgrade(upgrade::Version::V1)
            .authenticate(noise::NoiseConfig::xx(noise_keys).into_authenticated())
            .multiplex(mux_config)
            .boxed()
    };

    let mut swarm = {
        let mut behaviour = NodeStatusBehaviour::new().await?;
        SwarmBuilder::new(transport, behaviour, peer_id)
            .executor(Box::new(|fut| {
                tokio::spawn(fut);
            }))
            .build()
    };

    swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;
    swarm.listen_on("/ip4/0.0.0.0/tcp/0/ws".parse()?)?;
    let parsed_addr = MINA_PEER_ADDR.parse().unwrap();
    println!("Connecting to mina node {} ... ", MINA_PEER_ADDR);
    match swarm.dial_addr(parsed_addr) {
        Ok(_) => {
            println!("dial ok");
        }
        Err(e) => bail!("Fail to dail: {}", e),
    };
    loop {
        match swarm.select_next_some().await {
            SwarmEvent::NewListenAddr { address, .. } => println!("Listening on {:?}", address),
            SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                println!("Connected to {}", peer_id);
                swarm
                    .behaviour_mut()
                    .request_response
                    .send_request(&peer_id, NodeStatusRequest);
            }
            // SwarmEvent::Behaviour(MdnsEvent::Discovered(peers)) => {
            //     for (peer, addr) in peers {
            //         println!("discovered {} {}", peer, addr);
            //     }
            // }
            // SwarmEvent::Behaviour(MdnsEvent::Expired(expired)) => {
            //     for (peer, addr) in expired {
            //         println!("expired {} {}", peer, addr);
            //     }
            // }
            _ => {}
        }
    }
    // tokio::time::sleep(Duration::from_secs(60 * 60 * 24)).await;
    Ok(())
}

#[derive(NetworkBehaviour)]
#[behaviour(event_process = true)]
// #[behaviour(out_event = "NodeStatusEvent")]
struct NodeStatusBehaviour {
    request_response: RequestResponse<NodeStatusCodec>,
    mdns: Mdns,
}

impl NodeStatusBehaviour {
    async fn new() -> anyhow::Result<Self> {
        let mdns = Mdns::new(Default::default()).await?;
        let mut config = RequestResponseConfig::default();
        config.set_request_timeout(Duration::from_secs(60));
        Ok(Self {
            mdns,
            request_response: RequestResponse::new(
                NodeStatusCodec,
                std::iter::once((NodeStatusProtocol, ProtocolSupport::Full)),
                config,
            ),
        })
    }
}

impl NetworkBehaviourEventProcess<MdnsEvent> for NodeStatusBehaviour {
    fn inject_event(&mut self, event: MdnsEvent) {
        match event {
            MdnsEvent::Discovered(list) => {
                for (peer, _) in list {
                    println!("Peer discovered: {}", peer);
                }
            }
            MdnsEvent::Expired(list) => {
                for (peer, _) in list {
                    if !self.mdns.has_node(&peer) {
                        println!("Peer expired: {}", peer);
                    }
                }
            }
        }
    }
}

impl NetworkBehaviourEventProcess<RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>>
    for NodeStatusBehaviour
{
    fn inject_event(&mut self, event: RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>) {
        println!("RequestResponseEvent: {:?}", event);
    }
}

#[derive(Debug, Clone)]
struct NodeStatusProtocol;

impl ProtocolName for NodeStatusProtocol {
    fn protocol_name(&self) -> &[u8] {
        b"/mina/node-status"
        // b"/mytest"
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
        println!("read_request");
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
        println!("read_response: {}", json);
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
        println!("write_request");
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
        println!("write_response: {}", json);
        io.write_all(json.as_bytes()).await?;
        io.close().await?;
        Ok(())
    }
}

// #[derive(Debug)]
// enum NodeStatusEvent {
//     RequestResponse(RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>),
//     Mdns(MdnsEvent),
// }

// impl From<RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>> for NodeStatusEvent {
//     fn from(event: RequestResponseEvent<NodeStatusRequest, NodeStatusResponse>) -> Self {
//         println!("RequestResponseEvent0: {:?}", event);
//         NodeStatusEvent::RequestResponse(event)
//     }
// }

// impl From<MdnsEvent> for NodeStatusEvent {
//     fn from(event: MdnsEvent) -> Self {
//         println!("MdnsEvent: {:?}", event);
//         NodeStatusEvent::Mdns(event)
//     }
// }

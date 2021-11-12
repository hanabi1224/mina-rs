# Prerequisites
```
cargo install wasm-pack
```

# Commands
- Start [relay server](https://github.com/ChainSafe/mina-rs/tree/hl/playground/playground/mina-p2p-go-demo)

- Update index.ts with correct relay server address

    ```
    yarn
    yarn build
    yarn start
    ```

# Notes

- RSA identity is [not supported](https://github.com/libp2p/rust-libp2p/blob/adcfdc0750aca887e8e67ae241358bd30727bef6/core/src/identity.rs#L70) in wasm build, use Ed25519 or Secp256k1 instead

    ```rust
    #[derive(Clone)]
    pub enum Keypair {
        /// An Ed25519 keypair.
        Ed25519(ed25519::Keypair),
        #[cfg(not(target_arch = "wasm32"))]
        /// An RSA keypair.
        Rsa(rsa::Keypair),
        /// A Secp256k1 keypair.
        #[cfg(feature = "secp256k1")]
        Secp256k1(secp256k1::Keypair),
    }
    ```
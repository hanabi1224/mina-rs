import 'regenerator-runtime/runtime'
import wasmUrl from "./pkg/wasm_bg.wasm"
import { connect, wasm_test, wasm_test_async, init } from "./pkg/wasm"

async function main() {
    await init(await fetch(wasmUrl as any))
    console.log(`wasm_test: ${wasm_test()}`)
    console.log(`wasm_test_async: ${await wasm_test_async()}`)
    let connected = await connect('/ip4/127.0.0.1/tcp/36795/ws/p2p/12D3KooWETSQx1VDh1xoq1rwAaYFzzt4KGvXnKVEquvh2m6G64Ge')
    console.log(`Connected to relay: ${connected}`)
    connected = await connect('/ip4/127.0.0.1/tcp/36795/ws/p2p/12D3KooWETSQx1VDh1xoq1rwAaYFzzt4KGvXnKVEquvh2m6G64Ge/p2p-circuit/p2p/12D3KooWSxxCtzRLfUzoxgRYW9fTKWPUujdvStuwCPSPUN3629mb')
    console.log(`Connected to mina via relay: ${connected}`)
}

main()

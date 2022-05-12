initSidebarItems({"macro":[["impl_mina_enum_json_serde","Implement list tagged enum json serde format for the given enum, with another convertible enum which is externally tagged"],["impl_mina_enum_json_serde_with_option","Implement list tagged enum json serde format for the given enum, with another convertible enum which is externally tagged and extra serde options"],["impl_strconv_via_json","Implements [std::str::FromStr] and [std::fmt::Display] by implementing [TryFrom] between given type and string types via its corresponding json serialization type which is convertible from / to json with single unnamed string field."]],"mod":[["account","A Mina account record and supporting types This isn’t sent over the network but is serialized and stored in kv stores, so to be compatible with those we need to support these types"],["blockchain_state","Types related to the Blockchain State"],["bulletproof_challenges","Types that capture serialized bullet proof challenges and proofs"],["common","Some basic versioned types used throughout"],["consensus_state","Types and funcions related to the Mina consensus state"],["delta_transition_chain_proof","Delta transition chain proof structures and functions"],["epoch_data","Types and functions related to the EpochData structure"],["errors","Types that represent errors in mina serialization and deserialization"],["external_transition","Mina ExternalTransition"],["field_and_curve_elements","Versioned types that represent finite field and elliptic curve elements, and collections thereof"],["global_slot","Structure of a global slot"],["json","json serialization types for the Mina protocol"],["macros","Heper macros for type conversions"],["opening_proof","The opening proof used by the protocol state proof"],["proof_evaluations","Proof evaluations used by the protocol state proof"],["proof_messages","Proof messages used by the protocol state proof"],["protocol_constants","Types related to the Mina protocol state"],["protocol_state","Types related to the Mina protocol state"],["protocol_state_body","Types related to the Mina protocol state"],["protocol_state_proof","Module containing the components of a protocol state proof"],["protocol_version","Protocol version structure"],["signatures","Signatures and public key types"],["snark_work","Types related to the Transaction Snark Work"],["staged_ledger_diff","In this context a diff refers to a difference between two states of the blockchain. In this case it is between the current state and the proposed next state."],["v1","Version 1 serialization types for the Mina protocol"],["version_bytes","All human readable values (e.g base58 encoded hashes and addresses) implement the Base58Checked encoding https://en.bitcoin.it/wiki/Base58Check_encoding"]],"trait":[["BinProtSerializationType","This trait annotates a given type its corresponding bin-prot serialization type,"],["JsonSerializationType","This trait annotates a given type its corresponding json serialization type, and provide utility functions to easily convert between them"]]});
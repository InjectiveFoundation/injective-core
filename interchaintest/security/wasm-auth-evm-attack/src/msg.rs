use cosmwasm_schema::cw_serde;
use prost::{EncodeError, Message};
use prost_types;
use rlp::RlpStream;
use bytes::BytesMut;


#[cw_serde]
pub struct InstantiateMsg {}

#[cw_serde]
pub enum ExecuteMsg {
    SubmitAuthzGrant {
        grantee: String,
    },
    SubmitAuthzExec {
        granter: String,
    },
}

// Protobuf message definitions for authz
#[derive(Clone, PartialEq, Message)]
pub struct MsgGrant {
    #[prost(string, tag = "1")]
    pub granter: String,
    #[prost(string, tag = "2")]
    pub grantee: String,
    #[prost(message, optional, tag = "3")]
    pub grant: Option<Grant>,
}

#[derive(Clone, PartialEq, Message)]
pub struct Grant {
    #[prost(message, optional, tag = "1")]
    pub authorization: Option<prost_types::Any>,
    #[prost(message, optional, tag = "2")]
    pub expiration: Option<prost_types::Timestamp>,
}

#[derive(Clone, PartialEq, Message)]
pub struct GenericAuthorization {
    #[prost(string, tag = "1")]
    pub msg: String,
}

#[derive(Clone, PartialEq, Message)]
pub struct MsgExec {
    #[prost(string, tag = "1")]
    pub grantee: String,
    
    #[prost(message, repeated, tag = "2")]
    pub msgs: Vec<prost_types::Any>,
}

#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, Message)]
pub struct MsgEthereumTx {
    #[prost(bytes = "vec", tag = "5")]
    pub from: Vec<u8>,
    #[prost(bytes = "vec", tag = "6")]
    pub raw: Vec<u8>,
}

pub struct LegacyTransaction {
    pub nonce: u64,
    pub gas_price: u64,
    pub gas_limit: u64,
    pub to: Option<[u8; 20]>,
    pub value: u64,
    pub data: Vec<u8>,
}

pub fn rlp_encode(tx: &LegacyTransaction, r: &[u8], s: &[u8], v: u64) -> BytesMut {
    let mut stream = RlpStream::new();

    stream.begin_list(9);
    stream.append(&tx.nonce);
    stream.append(&tx.gas_price);
    stream.append(&tx.gas_limit);
    match tx.to {
        Some(ref to_addr) => stream.append(&to_addr.to_vec()),
        None => stream.append(&""),
    };
    stream.append(&tx.value);
    stream.append(&tx.data);
    stream.append(&v);
    stream.append(&r);
    stream.append(&s);
    stream.out()
}

pub fn pb_encode_message<M: Message>(message: &M) -> Result<Vec<u8>, EncodeError> {
    let mut buf = Vec::new();
    Message::encode(message, &mut buf)?;
    Ok(buf)
}
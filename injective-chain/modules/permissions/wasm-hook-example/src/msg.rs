use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Coin, CosmosMsg};
use injective_cosmwasm::InjectiveMsgWrapper;

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    UpdateOwner {
        new_owner: String,
    },
    ExecuteMsgs {
        msgs: Vec<CosmosMsg<InjectiveMsgWrapper>>,
    },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(Addr)]
    Owner {},
    #[returns(Addr)]
    SendRestriction {
        from_address: Addr,
        to_address: Addr,
        action: String,
        amounts: Vec<Coin>,
    },
}

#[cw_serde]
pub struct MigrateMsg {}

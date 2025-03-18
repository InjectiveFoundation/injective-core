package helpers

type WasmCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type IbcHooksQueryMsg struct {
	GetCount      *IbcHooksGetCountQuery      `json:"get_count,omitempty"`
	GetTotalFunds *IbcHooksGetTotalFundsQuery `json:"get_total_funds,omitempty"`
}

type WasmCounterQueryMsg struct {
	GetCount CounterGetCountQuery `json:"get_count,omitempty"`
}

type IbcHooksGetTotalFundsQuery struct {
	Addr string `json:"addr"`
}

type IbcHooksGetTotalFundsResponse struct {
	Data *IbcHooksGetTotalFundsObj `json:"data"`
}

type IbcHooksGetTotalFundsObj struct {
	TotalFunds []WasmCoin `json:"total_funds"`
}

type IbcHooksGetCountQuery struct {
	Addr string `json:"addr"`
}

type IbcHooksGetCountResponse struct {
	Data IbcHooksGetCountObj `json:"data"`
}

type IbcHooksGetCountObj struct {
	Count int64 `json:"count"`
}

type CounterGetCountQuery struct{}

type CounterGetCountResponse struct {
	Data CounterGetCountObj `json:"data"`
}

type CounterGetCountObj struct {
	Count int64 `json:"count"`
}

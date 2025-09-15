package types

import (
	"github.com/cosmos/gogoproto/proto"
)

func (d *Downtime) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Downtime_value, data, "Downtime")
	if err != nil {
		return err
	}
	*d = Downtime(value)
	return nil
}

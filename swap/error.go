package swap

import "encoding/json"

var (
	ErrInvalidParams         = &Error{10001, "invalid params"}
	ErrInvalidLiquidityPrice = &Error{20001, "invalid liquidity price"}
)

type Error struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func (e *Error) Error() string {
	eb, _ := json.Marshal(e)
	return string(eb)
}

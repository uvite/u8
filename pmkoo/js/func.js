
import ta from 'k6/x/ta'

export function genv(p){

	return ta.atr(close,high,low,p)
}


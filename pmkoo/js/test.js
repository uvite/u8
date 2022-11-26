import http from 'k6/http';

import {sleep} from 'k6'
import ta from 'k6/x/ta'

exports.options = {setupTimeout: "10s", teardownTimeout: "10s"};

export function setup() {

	console.log("12343333")
	const res = http.get('https://httpbin.test.k6.io/get');
	//
	// console.log(res.json())
	return {data: res.json()};
}

export function teardown(data) {
	console.log(JSON.stringify(data));
	console.log("tear");

}

export default function (data) {
	//console.log(data)


	while (true) {

		// 	let sma=ta.sma(close,14)
		// console.log("funck",close.last());
		// console.log("sma:",sma.last());
		// 	let sma=ta.sma(close,14)

		  let g = ta.dmi(  14, 14)
		 console.log(close.tail(3).reverse())
		  console.log(g.getADX().index(0),g.getADX().index(1),g.getADX().index(2))
		  console.log(g.getDIPlus().index(0),g.getDIPlus().index(1),g.getDIPlus().index(2))
		  console.log(g.getDIMinus().index(0),g.getDIMinus().index(1),g.getDIMinus().index(2))
		// // console.log(g.plus.last())
		// console.log(g.minus.tail(3).reverse())
		// console.log(g.adx.tail(3).reverse())

		sleep(30)
	}
}

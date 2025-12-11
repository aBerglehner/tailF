#!/usr/bin/env bash

first="Karen believed all traffic laws should be obeyed by all except herself"
second="It's never been my responsibility to glaze the donuts"
third="Pat ordered a ghost pepper pie"
fourth="The snow-covered path was no help in finding his way out of the back-country"
five="You have every right to be angry, but that doesn't give you the right to be mean"

lines=$(wc -l <"test.txt")

if [[ $lines -gt 2000 ]]; then
	for i in {1..50}; do
		yes ${first} | head -n 1 >test.txt
		yes ${second} | head -n 1 >>test.txt
		yes ${third} | head -n 1 >>test.txt
		yes ${fourth} | head -n 1 >>test.txt
		yes ${five} | head -n 1 >>test.txt
	done
fi

for i in {1..50}; do
	sleep 1.5
	yes ${first} | head -n 1 >>test.txt
	sleep 1.5
	yes ${second} | head -n 1 >>test.txt
	sleep 1.5
	yes ${third} | head -n 1 >>test.txt
	sleep 1.5
	yes ${fourth} | head -n 1 >>test.txt
	sleep 1.5
	yes ${five} | head -n 1 >>test.txt
done

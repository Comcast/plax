// collatz returns the Collatz sequence for its input.
//
// See https://en.wikipedia.org/wiki/Collatz_conjecture.
function collatz(n,steps) {
    if (!steps) {
	steps = [];
    }
    
    steps.push(n)
    
    if (n <= 1) {
	return steps;
    }

    if (n % 2 == 0) {
	return collatz(n/2,steps);
    } else {
	return collatz(3*n+1,steps);
    }
}

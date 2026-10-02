package collect

import "math"

// atLeast is log P(X >= n) and P(X < n) for X ~ Poisson(mu): the chance a key
// wanted n times has come n or more times once mu of it are expected. It is
// the regularized lower incomplete gamma P(n, mu), by its series below n+1
// and its continued fraction above (Numerical Recipes, 6.2), each in a form
// that stays exact far into its tail.
func atLeast(n, mu float64) (logP, q float64) {
	if mu <= 0 {
		return math.Inf(-1), 1
	}
	lg, _ := math.Lgamma(n)
	front := n*math.Log(mu) - mu - lg
	if mu < n+1 {
		sum, term := 1/n, 1/n
		for i := 1; i < 10000; i++ {
			term *= mu / (n + float64(i))
			sum += term
			if term < sum*1e-15 {
				break
			}
		}
		logP = front + math.Log(sum)
		return logP, -math.Expm1(logP)
	}
	const tiny = 1e-300
	b := mu + 1 - n
	c := 1 / tiny
	d := 1 / b
	h := d
	for i := 1; i < 10000; i++ {
		an := -float64(i) * (float64(i) - n)
		b += 2
		d = an*d + b
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = b + an/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < 1e-15 {
			break
		}
	}
	q = math.Exp(front + math.Log(h))
	return math.Log1p(-q), q
}

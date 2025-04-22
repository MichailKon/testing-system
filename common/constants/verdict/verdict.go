package verdict

type Verdict string

// The list is probably not complete
const (
	OK Verdict = "OK" // OK
	PT Verdict = "PT" // Partial solution

	WA Verdict = "WA" // Wrong answer
	WR Verdict = "WR" // Wrong (used only for interactive problems)

	RT Verdict = "RT" // Runtime error
	ML Verdict = "ML" // Memory limit
	TL Verdict = "TL" // Time limit
	WL Verdict = "WL" // Wall time limit
	SE Verdict = "SE" // Security violation

	CE Verdict = "CE" // Compilation error
	CD Verdict = "CD" // Compiled

	CF Verdict = "CF" // Check failed
	SK Verdict = "SK" // Skipped

	RU Verdict = "RU" // Running
)

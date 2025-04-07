package verdict

type Verdict string

// The list is probably not complete
const (
	OK Verdict = "OK" // OK
	PT Verdict = "PT" // Partial solution

	WA Verdict = "WA" // Wrong answer
	//PE Verdict = "PE" // Presentation error, we will treat it as Wrong Answer

	RT Verdict = "RT" // Runtime error
	ML Verdict = "ML" // Memory limit
	TL Verdict = "TL" // Time limit
	WL Verdict = "WL" // Wall time limit
	SE Verdict = "SE" // Security violation

	CE Verdict = "CE" // Compilation error
	CD Verdict = "CD" // Compiled

	CF Verdict = "CF" // Check failed

	QD Verdict = "QD" // Queued
	CL Verdict = "CL" // Compiling
	RU Verdict = "RU" // Running
)

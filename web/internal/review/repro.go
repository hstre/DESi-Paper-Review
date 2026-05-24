package review

import (
	"regexp"
	"strings"
)

// ReproRiskTypes enumerates all reproducibility-risk categories.
var ReproRiskTypes = []string{
	"missing_data", "missing_code", "missing_baselines",
	"missing_parameters", "unclear_dataset", "unsupported_metrics",
}

var sizedDatasetRe = regexp.MustCompile(
	`\b\d[\d,\.]*\s*(?:samples|examples|images|documents|sentences|instances|rows|records)\b`)

func hasAny(low string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(low, s) {
			return true
		}
	}
	return false
}

// DetectReproducibilityRisks scans the full paper text deterministically.
func DetectReproducibilityRisks(text string) []ReproRisk {
	low := strings.ToLower(text)
	risks := []ReproRisk{}

	if !hasAny(low, "github", "gitlab", "code is available", "code available",
		"source code", "open-source", "open source", "code repository",
		"implementation is available", "zenodo") {
		risks = append(risks, ReproRisk{
			RiskType: "missing_code",
			Detail:   "No link or statement indicating the source code is available for reproduction.",
		})
	}

	if !hasAny(low, "data is available", "data are available", "data available",
		"dataset is available", "dataset available", "publicly available",
		"supplementary data", "data repository", "zenodo") {
		risks = append(risks, ReproRisk{
			RiskType: "missing_data",
			Detail:   "No statement that the underlying data are available for inspection.",
		})
	}

	if !hasAny(low, "baseline", "compared to", "compared with", "comparison",
		"state-of-the-art", "state of the art", "prior work", "we compare") {
		risks = append(risks, ReproRisk{
			RiskType: "missing_baselines",
			Detail:   "No baseline or comparison against prior work is described.",
		})
	}

	if !hasAny(low, "hyperparameter", "learning rate", "epochs", "batch size",
		"random seed", "parameter settings", "training details",
		"we set", "configuration") {
		risks = append(risks, ReproRisk{
			RiskType: "missing_parameters",
			Detail:   "No hyperparameters, seeds, or training/configuration details are reported.",
		})
	}

	mentionsDataset := hasAny(low, "dataset", "data set", "corpus")
	named := hasAny(low, "imagenet", "cifar", "mnist", "glue", "squad",
		"wikitext", "coco", "penn treebank", "wmt")
	sized := sizedDatasetRe.MatchString(low)
	switch {
	case mentionsDataset && !(named || sized):
		risks = append(risks, ReproRisk{
			RiskType: "unclear_dataset",
			Detail:   "A dataset is mentioned but not identified by name, size, or construction.",
		})
	case !mentionsDataset:
		risks = append(risks, ReproRisk{
			RiskType: "unclear_dataset",
			Detail:   "No dataset is identified anywhere in the paper.",
		})
	}

	hasMetricWords := hasAny(low, "accuracy", "f1", "precision", "recall",
		"auc", "score", "performance", "%", "error rate")
	hasMetricMethod := hasAny(low, "evaluation protocol", "we measure",
		"computed as", "cross-validation", "cross validation", "test set",
		"held-out", "held out", "confidence interval", "standard deviation",
		"p-value", "p <", "significance test")
	if hasMetricWords && !hasMetricMethod {
		risks = append(risks, ReproRisk{
			RiskType: "unsupported_metrics",
			Detail:   "Performance metrics are reported without a described measurement protocol or variance/significance.",
		})
	}

	return risks
}

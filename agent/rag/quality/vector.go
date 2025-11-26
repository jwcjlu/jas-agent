package quality

import (
	"fmt"
	"math"
)

// ValidateVector 验证向量质量
func ValidateVector(vector []float32) error {
	if len(vector) == 0 {
		return fmt.Errorf("empty vector")
	}

	// 检查零向量
	if isZeroVector(vector) {
		return fmt.Errorf("zero vector")
	}

	// 检查 NaN 或 Inf
	if hasInvalidValues(vector) {
		return fmt.Errorf("vector contains NaN or Inf values")
	}

	// 检查向量是否归一化（可选）
	// norm := vectorNorm(vector)
	// if norm < 0.01 || norm > 100 {
	// 	return fmt.Errorf("vector norm out of range: %f", norm)
	// }

	return nil
}

// isZeroVector 检查是否为零向量
func isZeroVector(vector []float32) bool {
	for _, v := range vector {
		if math.Abs(float64(v)) > 1e-6 {
			return false
		}
	}
	return true
}

// hasInvalidValues 检查是否包含无效值（NaN、Inf）
func hasInvalidValues(vector []float32) bool {
	for _, v := range vector {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return true
		}
	}
	return false
}

// vectorNorm 计算向量范数（L2 范数）
func vectorNorm(vector []float32) float64 {
	var sum float64
	for _, v := range vector {
		sum += float64(v * v)
	}
	return math.Sqrt(sum)
}

// NormalizeVector 归一化向量（L2 归一化）
func NormalizeVector(vector []float32) []float32 {
	norm := vectorNorm(vector)
	if norm < 1e-9 {
		// 零向量，返回原向量
		return vector
	}

	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = float32(float64(v) / norm)
	}
	return normalized
}

// VectorSimilarity 计算两个向量的相似度（归一化后）
func VectorSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	// 归一化
	aNorm := NormalizeVector(a)
	bNorm := NormalizeVector(b)

	// 计算余弦相似度（归一化向量的点积）
	var dotProduct float64
	for i := range aNorm {
		dotProduct += float64(aNorm[i] * bNorm[i])
	}

	return dotProduct
}

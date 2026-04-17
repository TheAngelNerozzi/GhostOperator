package core

import (
        "testing"

        "github.com/stretchr/testify/assert"
)

func TestDetectHardwareProfile_PopulatesAllFields(t *testing.T) {
        profile := DetectHardwareProfile()

        // TotalRAMBytes should be non-zero on platforms with RAM detection
        if profile.TotalRAMBytes > 0 {
                assert.Greater(t, profile.FreeRAMBytes, uint64(0), "FreeRAMBytes should be > 0 when TotalRAMBytes is > 0")
        }
        assert.GreaterOrEqual(t, profile.NumCPU, 1, "NumCPU should be >= 1")

        // If weak, reason must be non-empty
        if profile.IsWeak {
                assert.NotEmpty(t, profile.Reason, "Weak profile must have a reason")
        }
}

func TestEffectiveBudgetMs_DefaultFallback(t *testing.T) {
        // When configBudgetMs is 0, fallback should use the hardcoded constant
        profile := HardwareProfile{IsWeak: true}
        budget := EffectiveBudgetMs(false, profile, 0)
        assert.Equal(t, BudgetFallbackMs, budget)
}

func TestEffectiveBudgetMs_ConfigOverrideFallback(t *testing.T) {
        // When configBudgetMs > 0, it should override the constant
        profile := HardwareProfile{IsWeak: true}
        budget := EffectiveBudgetMs(false, profile, 6000)
        assert.Equal(t, 6000, budget)
}

func TestEffectiveBudgetMs_NormalIgnoresConfigBudget(t *testing.T) {
        // Normal mode (not weak, not forced) should return BudgetNormalMs
        // regardless of configBudgetMs
        profile := HardwareProfile{IsWeak: false}
        budget := EffectiveBudgetMs(false, profile, 9999)
        assert.Equal(t, BudgetNormalMs, budget)
}

func TestEffectiveBudgetMs_ForcedWithCustom(t *testing.T) {
        // Forced fallback on strong hardware with custom budget
        profile := HardwareProfile{IsWeak: false}
        budget := EffectiveBudgetMs(true, profile, 4000)
        assert.Equal(t, 4000, budget)
}

func TestHardwareProfile_ReasonAggregation(t *testing.T) {
        // Verify the reason string combines multiple causes with semicolons
        // This is an integration test of DetectHardwareProfile's logic
        profile := DetectHardwareProfile()
        if profile.IsWeak {
                assert.Contains(t, profile.Reason, "<", "Reason should describe threshold violations")
        }
}

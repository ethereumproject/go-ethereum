package core

import (
	"math/big"
	"testing"
)

func TestBombWait(t *testing.T) {
	config := NewChainConfig()

	if (config.IsHomestead(big.NewInt(10000))) {
		t.Errorf("Unexpected for %d", 10000)
	}

	if (!config.IsHomestead(big.NewInt(1920000))) {
		t.Errorf("Expected for %d", 1920000)
	}
	if (!config.IsHomestead(big.NewInt(2325166))) {
		t.Errorf("Expected for %d", 2325166)
	}
	if (!config.IsHomestead(big.NewInt(3000000))) {
		t.Errorf("Expected for %d", 3000000)
	}
	if (!config.IsHomestead(big.NewInt(3000001))) {
		t.Errorf("Expected for %d", 3000001)
	}
	if (!config.IsHomestead(big.NewInt(4000000))) {
		t.Errorf("Expected for %d", 3000000)
	}
	if (!config.IsHomestead(big.NewInt(5000000))) {
		t.Errorf("Unexpected for %d", 5000000)
	}
	if (!config.IsHomestead(big.NewInt(5000001))) {
		t.Errorf("Unexpected for %d", 5000001)
	}
}

func TestBombDelay(t *testing.T) {
	config := NewChainConfig()

	if (config.IsGotham(big.NewInt(1920000))) {
		t.Errorf("Unexpected for %d", 1920000)
	}

	if (config.IsGotham(big.NewInt(2325166))) {
		t.Errorf("Unexpected for %d", 2325166)
	}

	if (!config.IsGotham(big.NewInt(3000000))) {
		t.Errorf("Expected for %d", 3000000)
	}
	if (!config.IsGotham(big.NewInt(3000001))) {
		t.Errorf("Expected for %d", 3000001)
	}
	if (!config.IsGotham(big.NewInt(4000000))) {
		t.Errorf("Expected for %d", 3000000)
	}

	if (config.IsGotham(big.NewInt(5000000))) {
		t.Errorf("Unexpected for %d", 5000000)
	}
	if (config.IsGotham(big.NewInt(5000001))) {
		t.Errorf("Unexpected for %d", 5000001)
	}
}

func TestBombExplode(t *testing.T) {
	config := NewChainConfig()

	if (config.IsExplosion(big.NewInt(1920000))) {
		t.Errorf("Unexpected for %d", 1920000)
	}

	if (config.IsExplosion(big.NewInt(2325166))) {
		t.Errorf("Unexpected for %d", 2325166)
	}

	if (config.IsExplosion(big.NewInt(3000000))) {
		t.Errorf("Unxpected for %d", 3000000)
	}
	if (config.IsExplosion(big.NewInt(3000001))) {
		t.Errorf("Unxpected for %d", 3000001)
	}
	if (config.IsExplosion(big.NewInt(4000000))) {
		t.Errorf("Unxpected for %d", 3000000)
	}

	if (!config.IsExplosion(big.NewInt(5000000))) {
		t.Errorf("Expected for %d", 5000000)
	}
	if (!config.IsExplosion(big.NewInt(5000001))) {
		t.Errorf("Expected for %d", 5000001)
	}

}

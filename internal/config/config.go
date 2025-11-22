package config

import "os"

type Config struct {
    CEMCPDFPath string
    NESPDFPath  string
}

// FromEnv builds a Config from environment variables, with sane defaults.
func FromEnv() Config {
    cemc := os.Getenv("CEMC_PDF_PATH")
    if cemc == "" {
        cemc = "/data/cemc_rates.pdf"
    }
    nes := os.Getenv("NES_PDF_PATH")
    if nes == "" {
        nes = "/data/nes_rates.pdf"
    }
    return Config{
        CEMCPDFPath: cemc,
        NESPDFPath:  nes,
    }
}

# Bernina Wi-Fi enabled USB gadget

## Directory structure

```
your-project/
├── .github/
│   └── workflows/
│       └── build.yml           # GitHub Actions for CI/CD
├── cmd/
│   └── your-app/
│       └── main.go            # Application entry point
├── internal/                  # Private application code
│   ├── config/
│   │   └── config.go
│   ├── handlers/
│   │   └── handlers.go
│   └── models/
│       └── models.go
├── pkg/                       # Public libraries (if any)
│   └── utils/
│       └── utils.go
├── scripts/                   # Build and deployment scripts
│   ├── build.sh
│   └── deploy.sh
├── assets/                    # Static resources
│   ├── images/
│   │   └── logo.png
│   └── templates/
│       └── index.html
├── docs/                      # Documentation
│   ├── api.md
│   └── setup.md
├── configs/                   # Configuration files
│   ├── config.yaml
│   └── config.example.yaml
├── test/                      # Integration tests
│   └── integration_test.go
├── build/                     # Build outputs (gitignored)
│   └── bin/
│       ├── your-app-linux-arm
│       └── your-app-darwin-amd64
├── .gitignore
├── .env.example
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```



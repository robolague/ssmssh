# SSM SSH

A beautiful, interactive terminal application for connecting to AWS EC2 instances via AWS Systems Manager Session Manager. No more remembering instance IDs or juggling multiple AWS profiles and regions!

## ✨ Features

- 🎨 **Beautiful TUI**: Interactive terminal interface with modern styling
- 🔍 **Smart Filtering**: Real-time search across profiles, regions, and instances
- 🏷️ **Instance Tags**: Preview instance tags and metadata before connecting
- ⚡ **Fast**: Efficiently loads and displays your AWS resources
- 🔒 **Secure**: Uses AWS SSM Session Manager (no SSH keys required)
- 🎯 **Multi-Profile**: Seamlessly switch between AWS profiles
- 🌍 **Multi-Region**: Browse instances across all AWS regions

## 🚀 Quick Start

### Prerequisites

1. **AWS CLI** installed and configured
2. **Session Manager Plugin** for AWS CLI
3. **Go 1.24+** (for building from source)

### Installation

#### Option 1: Download Binary (Recommended)

Download the latest release for your platform from the [Releases](https://github.com/robolague/ssmssh/releases) page.

#### Option 2: Build from Source

```bash
git clone https://github.com/robolague/ssmssh.git
cd ssmssh
go build -o ssmssh .
sudo mv ssmssh /usr/local/bin/
```

#### Option 3: Install with Go

```bash
go install github.com/robolague/ssmssh@latest
```

### Setup

1. **Install Session Manager Plugin**:
   ```bash
   # macOS
   curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/mac/sessionmanager-bundle.zip" -o "sessionmanager-bundle.zip"
   unzip sessionmanager-bundle.zip
   sudo ./sessionmanager-bundle/install -i /usr/local/sessionmanagerplugin -b /usr/local/bin/session-manager-plugin
   
   # Linux
   curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_64bit/session-manager-plugin.rpm" -o "session-manager-plugin.rpm"
   sudo yum install -y session-manager-plugin.rpm
   ```

2. **Configure AWS CLI** with your profiles:
   ```bash
   aws configure --profile my-profile
   ```

3. **Ensure EC2 instances have SSM agent** and proper IAM roles for Session Manager.

## 🎮 Usage

Simply run the application:

```bash
ssmssh
```

### Navigation

- **↑/↓ or j/k**: Navigate through options
- **Type**: Filter/search options in real-time
- **Enter**: Select current option
- **Esc, Ctrl+C, or Cmd+Q**: Exit

### Workflow

1. **Select AWS Profile**: Choose from your configured AWS profiles
2. **Select Region**: Pick the AWS region to browse
3. **Select Instance**: Choose the EC2 instance to connect to
4. **Connect**: Automatically starts an SSM session

## 🛠️ Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run with coverage
make test-coverage
```

### Available Make Targets

- `make build` - Build the application
- `make build-all` - Build for multiple platforms
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage
- `make install` - Install the application
- `make clean` - Clean build artifacts
- `make fmt` - Format code
- `make vet` - Run go vet
- `make quality` - Run fmt, vet, and test

## 🔧 Configuration

### AWS Profiles

SSM SSH automatically detects AWS profiles from your `~/.aws/credentials` file. No additional configuration needed!

### IAM Permissions

Your AWS profile needs the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeInstances",
                "ec2:DescribeRegions",
                "ssm:StartSession"
            ],
            "Resource": "*"
        }
    ]
}
```

## 🎨 Screenshots

The application features a beautiful terminal interface with:
- Color-coded selections
- Real-time filtering
- Instance tag previews
- Loading indicators
- Clean, modern styling

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Run tests: `make test`
5. Commit your changes: `git commit -am 'Add feature'`
6. Push to the branch: `git push origin feature-name`
7. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- Styled with [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Inspired by the need for better AWS instance management

## 🐛 Issues & Support

Found a bug or have a feature request? Please [open an issue](https://github.com/robolague/ssmssh/issues)!

---

**Made with ❤️ for the AWS community**

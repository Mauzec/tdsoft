# tdsoft – small telegram client scripts with GUI

## Overview

**tdsoft** provides a simple [Fyne](https://github.com/fyne-io/fyne) (Go) GUI for running Python [Pyrogram](https://docs.pyrogram.org/) scripts.

## Usage

* Requires **Go >= 1.21**
* Requires all **Python** dependencies installed (see `requirements.txt`)

Start the GUI

```bash
go run gui/cmd/main.go
# or
bash main.sh
```

## Logs

* UI logs are shown in the bottom panel
* Detailed logs are saved in `tdsoft.log`
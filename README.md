![License: CC BY-NC 4.0](https://img.shields.io/badge/License-CC%20BY--NC%204.0-lightgrey.svg)
#  🛡️ PROTECT-CHILDREN
protect-children is a Windows-based desktop application specifically designed to help parents monitor their children's digital activities. This program focuses on monitoring text input in specific applications to ensure children's safety from inappropriate content.

## ✨ Key Features

* **Smart Keyword Detection**: Automatically detects sensitive words or adult content. Suspicious words will be highlighted in red in the log.

* **Stealth Mode**: Runs silently in the background without disturbing your child's activities.

* **Auto-Run**: Option to run the application automatically as soon as the computer is turned on.

## 🛠️ How to Compile (Build)
To allow others to try or develop this program, they need to follow these steps:

##  🎯 Requirements:

* **Install Go (latest version).**

* **Install Node.js & NPM.**

* **Install Wails CLI**: go install github.com/wailsapp/wails/v2/cmd/wails@latest.

## 📦Build Steps:

* **Clone repo**: ```git clone https://github.com/username/protect-children.git.```

* **Install Frontend**: Go to the ```frontend``` folder and run ```npm install```.

* **Compile**: In the root directory (the folder containing ```wails.json```), run the command: ```wails build```
* **Result**: A ready-to-use ```.exe``` file will appear in the ```build/bin/``` folder.

## 🚀 Technology
* **Backend**: Go

* **Frontend**: Svelte (via Wails Framework)

### ⚠️ Disclaimer: This program is created purely for educational purposes and legal parental supervision. Any misuse of this program to violate the privacy of others without permission is the responsibility of each user.

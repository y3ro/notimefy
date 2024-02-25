# notimefy

Send email notifications when certain monthly hour thresholds are surpassed. 

## Installation

Assuming you have Go installed:

`go install github.com/y3ro/notimefy@latest`

You need to have `$HOME/go/bin` in your `PATH`.

## Usage

First you will need to create the configuration file `$HOME/.config/notimefy.json` (or specify your own filepath with the `-config` option).
Example contents:

```
{
    "kimaiUrl": "https://timetracking.domain.com",
    "kimaiUsername": "kimai-username",
    "kimaiPassword": "kimai-password",
    "hourThresholds": [40, 80, 120],
    "SMTPUsername": "user@email.com",
    "SMTPPassword": "email-password",
    "SMTPHost": "smtp.email.com",
    "SMTPPort": "587",
    "recipientEmail": "recipient@email.com"
}
```

Then, just run from anywhere:

```
notimefy <option>
```

I personally have a `crontab` entry to periodically run this app:

```
0 12,23 * * 1-6 notimefy
```

Avaliable options:

* `-reset-first`: Reset the app's state before running. Could be useful if the file `$HOME/.local/share/notimefy/<kimai_host>`, which is managed by the app, falls into a inconsistent state, mostly when changing the hour thresholds in the config file (you can manually delete the the app's state file in this case).
* `-config <filepath>`: Specifies the path to the configuration file. If not specified, the default configuration file is in `$HOME/.config/notimefy.json`. 

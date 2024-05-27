# kor
[1.1]: http://i.imgur.com/tXSoThF.png
[1]: https://twitter.com/TobiunddasMoe
This an adaption of Emoe's adaptation of tomnomnom's kxss tool with a different output format and some flags for custom headers and proxy. I didn't want to fork his whole Hacks-Repository so created my Own ;-)

It has also been adapted to check for Open Redirects instead of XSS.

All Credit for this Code goes to [Tomnomnom](https://github.com/tomnomnom/) and [Emoe](https://github.com/Emoe/)

## Output
Output Looks like this:
```
URL: https://www.**********.***/event_register.php?event=177 Param: event Unfiltered: [http://quas.sh http:/quas.sh]
```

## Installation
To install this Tool please use the following Command:
```
go install github.com/microphone-mathematics/kor@latest
```

## Usage
### Basic usage
To run this script use the following command:
```
echo "https://www.**********.***/event_register.php?event=177" | kor
```

### Custom Headers
```
echo "https://www.**********.***/event_register.php?event=177" | kor -header 'Cookie: JSESSIONID=xxxxxxxxxxxxxxx' -header 'Authorization: Bearer aaaaaaaaaaaaaaaaaa'
```

### Custom Proxy
```
echo "https://www.**********.***/event_register.php?event=177" | kor -proxy 'http://127.0.0.1:8080'
```

## Question
If you have an question you can create an Issue or ping me on [![alt text][1.1]][1]

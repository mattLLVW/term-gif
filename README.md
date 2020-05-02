![magic](/docs/e.xec.sh.png)

# Usage

```bash
curl e.xec.sh/anything_you_want
```

Search terms must be separated by an underscore.

Works with `curl` and `wget`:

```bash
curl e.xec.sh/magic
wget -qO- e.xec.sh/rainbow
```

## Example

[![asciicast](https://asciinema.org/a/qTVUtjVThqmrJadmLZHvmt7H1.svg)](https://asciinema.org/a/qTVUtjVThqmrJadmLZHvmt7H1)

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## Run locally

```bash
cp env.sample .env
go build main.go
./main
```
and connect to [localhost:9000/whatever_you_want](localhost:9000/whatever_you_want)


## Donations

If you like this project, consider donating:

via GitHub Sponsors, or

[![ko-fi](https://www.ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/J3J3173F6)


## Licence
[MIT](https://choosealicense.com/licenses/mit/)

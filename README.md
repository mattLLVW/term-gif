![magic](/static/img/e.xec.sh.png)

# Usage

```bash
curl e.xec.sh/<your_search_terms_separated_by_an_underscore>
```
or

```bash
curl e.xec.sh?url=<https://your_image_or_gif_url.gif>
```

## Example

```
curl e.xec.sh/spongebob_magic
curl "e.xec.sh?url=https://e.xec.sh/static/img/mgc.gif" # Any url as long as it's an image or a gif
```

You can also reverse the gif if you want, i.e:

```
curl "e.xec.sh/mind_blown?rev=true"
```

Or just display a preview image of the gif, i.e:

```
curl "e.xec.sh/wow?img=true"
```


Works with `curl` and `wget`:

```bash
curl e.xec.sh/magic
wget -qO- e.xec.sh/rainbow
```

![magic](/static/img/magic.gif)

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## Run locally

```bash
cp env.sample .env # and fill it with your api key.
docker-compose up --build
```
and connect to [localhost:9000/whatever_you_want](localhost:9000/whatever_you_want)


## Donations

If you like this project, consider donating:

via [GitHub Sponsors](https://github.com/sponsors/mattLLVW)

Bitcoin Lightning: 

![lightning](/static/img/address.png)

[![ko-fi](https://www.ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/J3J3173F6)


## Licence
[MIT](https://choosealicense.com/licenses/mit/)

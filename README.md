# GIF.XYZZY.RUN

# Usage

```bash
curl gif.xyzzy.run/<your_search_terms_separated_by_an_underscore>
```
or

```bash
curl gif.xyzzy.run?url=<https://your_image_or_gif_url.gif>
```

## Example

```
curl gif.xyzzy.run/spongebob_magic
curl "gif.xyzzy.run?url=https://gif.xyzzy.run/static/img/mgc.gif" # Any url as long as it's an image or a gif
```

You can also reverse the gif if you want, i.e:

```
curl "gif.xyzzy.run/mind_blown?rev=true"
```

Or just display a preview image of the gif, i.e:

```
curl "gif.xyzzy.run/wow?img=true"
```


Works with `curl` and `wget`:

```bash
curl gif.xyzzy.run/magic
wget -qO- gif.xyzzy.run/rainbow
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

[![ko-fi](https://www.ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/J3J3173F6)


## Licence
[MIT](https://choosealicense.com/licenses/mit/)

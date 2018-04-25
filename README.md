# `concur`
>create and synchronise tracks posted on various subreddits with spotify playlist

### Motivation
I have been a lurker on reddit for more than half a decade, and most of the new music discoveries come from either
[/r/music](https://reddit.com/r/music) or [/r/listentothis](https://reddit.com/r/listentothis), its a chore to listen 
to tracks there, find it on Spotify and then add them to a playlist. I initially tried to automate it with Zapier. 
Its an amazing tool for that, I had created a Zap that gets executed whenever there's a hot post in either of the 
subreddit and added that particular track in spotify if it exists, it was the easiest solution I could find but it
had two flaws, one being it only supports new posts rather than what had already been posted there and another flaw
being its cost. This is where this tiny app comes in, you can run it whenever you want up to many pages furthermore
you can also set up a lambda function on AWS and invoke it via cloudwatch to automate it like Zapier.

### Setup
You can install this via following methods:
```bash
# via go get
λ go get -u github.com/umayr/concur/cmd/concur

# via git
λ git clone github.com/umayr/concur
λ make
```

OR you can simply download prebuild binaries from [here](https://github.com/umayr/concur/releases).

### Usage
To get it working, you'd require a Client ID and Secret for Spotify. You can create an app from [here](https://beta.developer.spotify.com)
and provide a redirect URI which Spotify would be using to complete authentication process. Once you have Client ID and
Secret, set them as environment variables (`SPOTIFY_ID` and `SPOTIFY_SECRET` respectively) and do the following:
```bash
# make sure you have concur binary in your $PATH variable
λ concur -subreddit=music -pages=3 -redirect-uri='http://localhost:8080/callback' -playlist-id=4WftiOQe0gRuis2AfKF3VS
```
And that would be it.

Furthermore, if you have logs enabled via setting `DEBUG_CONCUR` environment variable, you can see verbose logs in your
terminal, you can extract refresh token from those logs and use that to skip the authentication part like this:
```bash
λ concur -subreddit=music,listentothis -pages=10 -refresh-token='<token>'
```
If you don't provide a playlist ID, then it would create a playlist for you. For command above, new playlist would be
named `Reddit Sync - /r/music /r/listentothis`.

### Todo
- create a server that does the same functionality while abstracting all the hassle

Feel free to suggest anything else. 

### License
MIT - [Umayr Shahid](mailto:umayr.shahid@gmail.com)

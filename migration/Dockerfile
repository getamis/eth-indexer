FROM ruby:2.3.0
MAINTAINER miya@am.is
ENV LANG C.UTF-8
ENV HOST localhost
ENV PORT 3306
ENV USERNAME root
ENV PASSWORD my-secret-pw
ENV DATABASE eth-db

RUN apt-get update && \
    apt-get install -y unixodbc-dev \
                       mysql-client \
                       --no-install-recommends && \
    rm -rf /var/lib/apt/lists/*

#Cache bundle install
WORKDIR /tmp
ADD ./Gemfile Gemfile
ADD ./Gemfile.lock Gemfile.lock
RUN bundle install

ADD ./Rakefile Rakefile
ADD ./db db

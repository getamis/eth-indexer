FROM ruby:2.3.0
ENV LANG C.UTF-8
ENV HOST localhost
ENV PORT 3306
ENV USERNAME root
ENV PASSWORD my-secret-pw
ENV DATABASE ethdb

RUN apt-get update && \
    apt-get install -y unixodbc-dev \
                       mysql-client \
                       freetds-dev \
                       build-essential \
                       patch \
                       ruby-dev \
                       zlib1g-dev \
                       liblzma-dev \
                       --no-install-recommends && \
    rm -rf /var/lib/apt/lists/*

# Cache bundle install
WORKDIR /tmp
ADD ./Gemfile Gemfile
ADD ./Gemfile.lock Gemfile.lock
RUN bundle install

ADD ./Rakefile Rakefile
ADD ./db db

RUN chmod -R 777 Rakefile && chmod -R 777 db

RUN addgroup --system --gid 699 app
RUN adduser --system --uid 699 --gid 699 --no-create-home --disabled-login app

USER app

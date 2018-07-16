####### Install libraries
```
bundle install --path=vendor/bundle
```

####### generate a new migration plan

```
bundle exec rake db:new_migration name=foo_bar_migration
```

####### modify migration script
```
vim db/migration/foo_bar_migration.rb
```

####### kick off mysql service via docker
```
docker run -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=my-secret-pw -e MYSQL_CHARSET=utf8 -e MYSQL_DATABASE=ethdb --name indexer-mysql mysql:5.7 --character-set-server=utf8 --collation-server=utf8_unicode_ci
```

####### run mysql migration to upgrade schema.rb
```
bundle exec rake db:migrate
```

####### do migration with docker
```
// build docker migration image
docker build -t quay.io/amis/indexer-db-migration .

// run migration
docker run -e RAILS_ENV=customized -e HOST=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' indexer-mysql) -e PORT=3306 -e DATABASE=indexer-db -e USERNAME=root -e PASSWORD=my-secret-pw quay.io/amis/indexer-db-migration bundle exec rake db:migrate
```

####### do rollback
```
docker run -e RAILS_ENV=customized -e HOST=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' indexer-mysql) -e PORT=3306 -e DATABASE=indexer-db -e USERNAME=root -e PASSWORD=my-secret-pw quay.io/amis/indexer-db-migration bundle exec rake db:rollback
```

####### check STATUS
```
docker run -e RAILS_ENV=customized -e HOST=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' indexer-mysql) -e PORT=3306 -e DATABASE=indexer-db -e USERNAME=root -e PASSWORD=my-secret-pw quay.io/amis/indexer-db-migration bundle exec rake db:migrate:status
```

reference: https://github.com/thuss/standalone-migrations

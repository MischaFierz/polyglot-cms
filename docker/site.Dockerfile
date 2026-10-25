FROM php:8.3-apache

RUN a2enmod rewrite \
 && sed -ri 's#/var/www/html#/var/www/site/public#g' /etc/apache2/sites-available/000-default.conf \
 && printf '<Directory /var/www/site/public>\n    AllowOverride All\n    Require all granted\n</Directory>\n' \
    > /etc/apache2/conf-enabled/site.conf

COPY site/ /var/www/site/
RUN mkdir -p /var/www/site/cache /var/www/site/public/uploads \
 && chown -R www-data /var/www/site/cache

ENV CMS_API_URL=http://api:8080 \
    CMS_CACHE_TTL=15 \
    CMS_THEME=default

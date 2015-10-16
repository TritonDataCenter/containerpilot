FROM 0x74696d/example-base

# add containerbuddy and all our configuration
RUN mkdir /opt/containerbuddy
ADD containerbuddy/* /opt/containerbuddy/

# add the Nginx configuration
ADD nginx/default.conf /etc/nginx/conf.d/default.conf

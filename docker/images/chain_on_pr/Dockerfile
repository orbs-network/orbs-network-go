FROM circleci/golang:1.12.9

# Install Node.js
COPY setup.sh ./setup.sh
RUN ./setup.sh
ENV NVM_DIR="/home/circleci/.nvm"
ENV NODE_VERSION="v10.14.1"

RUN . $NVM_DIR/nvm.sh && \
    nvm install $NODE_VERSION && \
    nvm alias default $NODE_VERSION && \
    nvm use default && \
    node -v

# Installing AWS cli
RUN sudo apt-get update && \ 
    sudo apt-get install -y python-dev && \
    sudo apt-get install -y python-pip && \
    sudo pip install awscli && \
    aws --version
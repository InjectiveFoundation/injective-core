# Dockerfile for evm contract deployment container
# Based on the injective-core image
FROM injectivelabs/injective-core:local

# Install basic tools and build dependencies
RUN apt update && apt install -y \
    git \
    bash \
    jq \
    python3 \
    python3-pip
    
# Install foundry (forge, cast, anvil, chisel)
ENV FOUNDRY_DIR=/root/.foundry
RUN curl -L https://foundry.paradigm.xyz | bash && \
    "$FOUNDRY_DIR/bin/foundryup" && \
    ln -s "$FOUNDRY_DIR/bin/forge" /usr/local/bin/forge && \
    ln -s "$FOUNDRY_DIR/bin/cast" /usr/local/bin/cast && \
    ln -s "$FOUNDRY_DIR/bin/anvil" /usr/local/bin/anvil && \
    ln -s "$FOUNDRY_DIR/bin/chisel" /usr/local/bin/chisel

# Install Node.js 20.18.3 via nvm and yarn
ENV NVM_DIR=/root/.nvm
ENV NODE_VERSION=20.18.3
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash && \
    bash -c "source $NVM_DIR/nvm.sh && nvm install $NODE_VERSION && nvm use $NODE_VERSION && npm i -g yarn@1.22.19" && \
    ln -s $NVM_DIR/versions/node/v$NODE_VERSION/bin/node /usr/local/bin/node && \
    ln -s $NVM_DIR/versions/node/v$NODE_VERSION/bin/npm /usr/local/bin/npm && \
    ln -s $NVM_DIR/versions/node/v$NODE_VERSION/bin/yarn /usr/local/bin/yarn

# Set working directory
WORKDIR /workspace    

# Keep container running
CMD ["tail", "-f", "/dev/null"]

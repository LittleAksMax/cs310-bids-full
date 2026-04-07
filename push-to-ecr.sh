
AWSACCID="$1"
IMAGE="$2"
VERSION="$3"
PLATFORM="${4:-linux/arm64}" # we default to ARM because dev machine is Apple Silicon

echo "Acc. ID: $AWSACCID -- Image: $IMAGE -- Vers.: $VERSION -- Platform: $PLATFORM"

aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin $AWSACCID.dkr.ecr.eu-west-2.amazonaws.com

docker build --platform $PLATFORM -t bidder/$IMAGE:$VERSION .

docker tag bidder/$IMAGE:$VERSION $AWSACCID.dkr.ecr.eu-west-2.amazonaws.com/bidder/$IMAGE:$VERSION

docker push $AWSACCID.dkr.ecr.eu-west-2.amazonaws.com/bidder/$IMAGE:$VERSION

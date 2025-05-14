#!/bin/bash

# Function to increment the patch version
increment_patch_version() {
  IFS='.' read -r major minor patch <<< "$1"
  patch=$((patch + 1))
  echo "$major.$minor.$patch"
}

# Ask for the current plugin version
read -p "Enter the current plugin version (e.g., 1.0.0): " current_version

# Increment the patch version
new_version=$(increment_patch_version "$current_version")
echo "New plugin version will be: $new_version"

# Ask for the GitLab personal access token
read -sp "Enter your GitLab PAT (Personal Access Token): " PATGITLABTOKEN
echo

# Build the plugin
echo "Building the plugin..."
go build -o gitlab-plugin
if [ $? -ne 0 ]; then
  echo "Error: Failed to build the plugin."
  exit 1
fi

# Move the built plugin to the plugins folder
echo "Moving the plugin to the plugins folder..."
mv ./gitlab-plugin ./plugins/gitlab-plugin
if [ $? -ne 0 ]; then
  echo "Error: Failed to move the plugin."
  exit 1
fi

# Create the SHA256 hash
echo "Generating SHA256 hash..."
SHA=$(sha256sum ./plugins/gitlab-plugin | awk '{print $1;}')
if [ -z "$SHA" ]; then
  echo "Error: Failed to generate SHA256 hash."
  exit 1
fi
echo "SHA256: $SHA"

# Deregister the old plugin version
echo "Deregistering old plugin version: $current_version..."
vault plugin deregister -version="v$current_version" secret gitlab
if [ $? -ne 0 ]; then
  echo "Error: Failed to deregister the old plugin version."
  exit 1
fi

# Register the new plugin version
echo "Registering new plugin version: $new_version..."
vault plugin register -command gitlab-plugin -sha256 "$SHA" -version "v$new_version" secret gitlab
if [ $? -ne 0 ]; then
  echo "Error: Failed to register the new plugin version."
  exit 1
fi

# Disable the old secret engine version
echo "Disabling old secret engine version..."
vault secrets disable gitlab
if [ $? -ne 0 ]; then
  echo "Error: Failed to disable the old secret engine."
  exit 1
fi

# Enable the new secret engine version
echo "Enabling new secret engine version..."
vault secrets enable -path gitlab gitlab
if [ $? -ne 0 ]; then
  echo "Error: Failed to enable the new secret engine."
  exit 1
fi

# Create backend config
echo "Creating backend config..."
vault write gitlab/config base_url="https://qual-git.service.migros.cloud" token="$PATGITLABTOKEN"
if [ $? -ne 0 ]; then
  echo "Error: Failed to create backend config."
  exit 1
fi

# List secret engines and check the version
echo "Listing secret engines to verify the version..."
vault secrets list -detailed
if [ $? -ne 0 ]; then
  echo "Error: Failed to list secret engines."
  exit 1
fi

echo "Plugin reload and testing steps completed successfully!"

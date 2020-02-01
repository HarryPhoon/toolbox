#!/bin/sh
#
# Copyright © 2018 – 2019 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


exec 3>/dev/null

arguments=""
assume_yes=false
base_toolbox_command=$(basename "$0" 2>&3)
base_toolbox_image=""
cgroups_version=""

# Based on the nameRegex value in:
# https://github.com/containers/libpod/blob/master/libpod/options.go
container_name_regexp="[a-zA-Z0-9][a-zA-Z0-9_.-]*"

environment=$(set)
environment_variables="COLORTERM \
        COLUMNS \
        DBUS_SESSION_BUS_ADDRESS \
        DBUS_SYSTEM_BUS_ADDRESS \
        DESKTOP_SESSION \
        DISPLAY \
        LANG \
        LINES \
        SHELL \
        SSH_AUTH_SOCK \
        TERM \
        TOOLBOX_PATH \
        VTE_VERSION \
        WAYLAND_DISPLAY \
        XDG_CURRENT_DESKTOP \
        XDG_DATA_DIRS \
        XDG_MENU_PREFIX \
        XDG_RUNTIME_DIR \
        XDG_SEAT \
        XDG_SESSION_DESKTOP \
        XDG_SESSION_ID \
        XDG_SESSION_TYPE \
        XDG_VTNR"
fgc=""

podman_command="podman"
registry="registry.fedoraproject.org"
registry_candidate="candidate-registry.fedoraproject.org"
release=""
release_default=""
spinner_animation="[>----] [=>---] [==>--] [===>-] [====>] [----<] [---<=] [--<==] [-<===] [<====]"
spinner_template="toolbox-spinner-XXXXXXXXXX"
tab="$(printf '\t')"
toolbox_command_path=""
toolbox_container=""
toolbox_container_default=""
toolbox_container_old_v1=""
toolbox_container_old_v2=""
toolbox_container_prefix_default=""
toolbox_image=""
toolbox_runtime_directory="$XDG_RUNTIME_DIR"/toolbox
user_id_real=$(id -ru 2>&3)
verbose=false
notty=false


LGC='\033[1;32m' # Light Green Color
NC='\033[0m' # No Color


has_prefix()
(
    str="$1"
    prefix="$2"

    ret_val=1

    case "$str" in
        "$prefix"* )
            ret_val=0
            ;;
        * )
            ret_val=1
            ;;
    esac

    return "$ret_val"
)


has_substring()
(
    haystack="$1"
    needle="$2"

    ret_val=1

    case "$haystack" in
        *"$needle"* )
            ret_val=0
            ;;
        * )
            ret_val=1
            ;;
    esac

    return "$ret_val"
)


is_integer()
{
    [ "$1" != "" ] && [ "$1" -eq "$1" ] 2>&3
    return "$?"
}


save_positional_parameters()
{
    for i; do
        printf "%s\\n" "$i" | sed "s/'/'\\\\''/g;1s/^/'/;\$s/\$/' \\\\/" 2>&3
    done
    echo " "
}


spinner_start()
(
    directory="$1"
    message="$2"

    if $verbose; then
        rm --force --recursive "$directory" 2>&3
        return 0
    fi

    if ! touch "$directory/spinner-start" 2>&3; then
        echo "$base_toolbox_command: unable to start spinner: spinner start file couldn't be created" >&2
        return 1
    fi

    printf "%s" "$message"
    tput civis 2>&3

    exec 4>"$directory/spinner-start"
    if ! flock 4 2>&3; then
        echo "$base_toolbox_command: unable to start spinner: spinner lock couldn't be acquired" >&2
        return 1
    fi

    (
        while [ -f "$directory/spinner-start" ]; do
            echo "$spinner_animation" | sed "s/ /\n/g" 2>&3 | while read -r frame; do
                if ! [ -f "$directory/spinner-start" ] 2>&3; then
                   break
                fi

                printf "%s" "$frame"

                frame_len=${#frame}
                i=0
                while [ "$i" -lt "$frame_len" ]; do
                    printf "\b"
                    i=$((i + 1))
                done

                sleep 1
            done
        done

        printf "\033[2K" # delete entire line regardless of cursor position
        printf "\r"
        tput cnorm 2>&3
    ) &

    return 0
)


spinner_stop()
(
    $verbose && return
    directory="$1"

    exec 4>"$directory/spinner-start"

    if ! rm "$directory/spinner-start" 2>&3; then
        echo "$base_toolbox_command: unable to stop spinner: spinner start file couldn't be removed" >&2
        return
    fi

    if ! flock 4 2>&3; then
        echo "$base_toolbox_command: unable to stop spinner: spinner lock couldn't be acquired" >&2
        return
    fi

    rm --force --recursive "$directory" 2>&3
)


ask_for_confirmation()
(
    default_response="$1"
    prompt="$2"
    ret_val=0

    while :; do
        printf "%s " "$prompt"
        read -r user_response

        if [ "$user_response" = "" ] 2>&3; then
            user_response="$default_response"
        else
            user_response=$(echo "$user_response" | tr "[:upper:]" "[:lower:]" 2>&3)
        fi

        if [ "$user_response" = "no" ] 2>&3 || [ "$user_response" = "n" ] 2>&3; then
            ret_val=1
            break
        elif [ "$user_response" = "yes" ] 2>&3 || [ "$user_response" = "y" ] 2>&3; then
            ret_val=0
            break
        fi
    done

    return "$ret_val"
)


container_name_is_valid()
(
    name="$1"

    echo "$name" | grep "^$container_name_regexp$" >/dev/null 2>&3
    return "$?"
)


container_start()
(
    container="$1"

    error_message=$( ($podman_command start "$container" >/dev/null) 2>&1)
    ret_val="$?"
    [ "$error_message" != "" ] 2>&3 && echo "$error_message" >&3

    if [ "$ret_val" -ne 0 ] 2>&3; then
        if echo "$error_message" | grep "use system migrate to mitigate" >/dev/null 2>&3; then
            echo "$base_toolbox_command: checking if 'podman system migrate' supports --new-runtime" >&3

            if ! ($podman_command system migrate --help 2>&3 | grep "new-runtime" >/dev/null 2>&3); then
                echo "$base_toolbox_command: container $container doesn't support cgroups v$cgroups_version" >&2
                echo "Update Podman to version 1.6.2 or newer." >&2
                return 1
            else
                echo "$base_toolbox_command: 'podman system migrate' supports --new-runtime" >&3

                oci_runtime_required="runc"
                [ "$cgroups_version" -eq 2 ] 2>&3 && oci_runtime_required="crun"

                echo "$base_toolbox_command: migrating containers to OCI runtime $oci_runtime_required" >&3

                if ! $podman_command system migrate --new-runtime "$oci_runtime_required" >/dev/null 2>&3; then
                    echo "$base_toolbox_command: failed to migrate containers to OCI runtime $oci_runtime_required" >&2
                    echo "Factory reset with: toolbox reset" >&2
                    echo "Try '$base_toolbox_command --help' for more information." >&2
                    return 1
                fi

                if ! $podman_command start "$container" >/dev/null 2>&3; then
                    echo "$base_toolbox_command: container $container doesn't support cgroups v$cgroups_version" >&2
                    echo "Factory reset with: toolbox reset" >&2
                    echo "Try '$base_toolbox_command --help' for more information." >&2
                    return 1
                fi
            fi
        else
            echo "$base_toolbox_command: failed to start container $container" >&2
            return 1
        fi
    fi

    return 0
)


copy_etc_profile_d_toolbox_to_container()
(
    container="$1"

    profile_d_lock="$toolbox_runtime_directory"/profile.d-toolbox.lock

    # shellcheck disable=SC2174
    if ! mkdir --mode 700 --parents "$toolbox_runtime_directory" 2>&3; then
        echo "$base_toolbox_command: unable to copy $toolbox_runtime_directory/toolbox.sh: runtime directory not created" >&2
        return 1
    fi

    exec 5>"$profile_d_lock"
    if ! flock 5 2>&3; then
        echo "$base_toolbox_command: unable to copy $toolbox_runtime_directory/toolbox.sh: copy lock not acquired" >&2
        return 1
    fi

    if ! [ -f "$toolbox_runtime_directory"/toolbox.sh ] 2>&3; then
        echo "$base_toolbox_command: $toolbox_runtime_directory/toolbox.sh not found" >&2
        return 0
    fi

    echo "$base_toolbox_command: copying $toolbox_runtime_directory/toolbox.sh to container $container" >&3

    if ! $podman_command exec \
                 --user root:root \
                 "$container" \
                 sh -c "cp $toolbox_runtime_directory/toolbox.sh /etc/profile.d" sh 2>&3; then
        echo "$base_toolbox_command: unable to copy $toolbox_runtime_directory/toolbox.sh to container $container" >&2
        return 1
    fi

    return 0
)


copy_etc_profile_d_toolbox_to_runtime_directory()
(
    profile_d_lock="$toolbox_runtime_directory"/profile.d-toolbox.lock

    if ! [ -f /etc/profile.d/toolbox.sh ] 2>&3; then
        echo "$base_toolbox_command: /etc/profile.d/toolbox.sh not found" >&2
        return 0
    fi

    # shellcheck disable=SC2174
    if ! mkdir --mode 700 --parents "$toolbox_runtime_directory" 2>&3; then
        echo "$base_toolbox_command: unable to copy /etc/profile.d/toolbox.sh: runtime directory not created" >&2
        return 1
    fi

    exec 5>"$profile_d_lock"
    if ! flock 5 2>&3; then
        echo "$base_toolbox_command: unable to copy /etc/profile.d/toolbox.sh: copy lock not acquired" >&2
        return 1
    fi

    echo "$base_toolbox_command: copying /etc/profile.d/toolbox.sh to $toolbox_runtime_directory" >&3

    if ! cp /etc/profile.d/toolbox.sh "$toolbox_runtime_directory" 2>&3; then
        echo "$base_toolbox_command: unable to copy /etc/profile.d/toolbox.sh to $toolbox_runtime_directory" >&2
        return 1
    fi

    return 0
)


create_enter_command()
(
    container="$1"

    if [ "$container" = "$toolbox_container_default" ] 2>&3; then
        echo "$base_toolbox_command enter"
    elif [ "$container" = "$toolbox_container_prefix_default-$release" ] 2>&3; then
        echo "$base_toolbox_command enter --release $release"
    else
        echo "$base_toolbox_command enter --container $container"
    fi
)


create_environment_options()
(
    columns=""
    lines=""

    if terminal_size=$(stty size 2>&3); then
        columns=$(echo "$terminal_size" | cut --delimiter " " --fields 2 2>&3)
        if ! is_integer "$columns"; then
            echo "$base_toolbox_command: failed to parse the number of columns from the terminal size" >&3
            columns=""
        fi

        lines=$(echo "$terminal_size" | cut --delimiter " " --fields 1 2>&3)
        if ! is_integer "$lines"; then
            echo "$base_toolbox_command: failed to parse the number of lines from the terminal size" >&3
            lines=""
        fi
    else
        echo "$base_toolbox_command: failed to read terminal size" >&3
    fi

    echo "$environment_variables" \
        | sed "s/ \+/\n/g" 2>&3 \
        | {
              environment_options=""
              echo "$base_toolbox_command: creating list of environment variables to forward" >&3
              value=""
              while read -r variable; do
                  if echo "$environment" | grep "^$variable" >/dev/null 2>&3; then
                      eval value="$""$variable"
                      echo "$base_toolbox_command: $variable=$value" >&3
                      environment_options="$environment_options --env=$variable=$value"
                  else
                      echo "$base_toolbox_command: $variable is unset" >&3
                  fi
              done

              if ! (echo "$environment_options" | grep COLUMNS >/dev/null 2>&3) && [ "$columns" != "" ] 2>&3; then
                  environment_options="$environment_options --env=COLUMNS=$columns"
              fi

              if ! (echo "$environment_options" | grep LINES >/dev/null 2>&3) && [ "$lines" != "" ] 2>&3; then
                  environment_options="$environment_options --env=LINES=$lines"
              fi

              environment_options=${environment_options#" "}
              echo "$base_toolbox_command: created options for environment variables to forward" >&3
              echo "$environment_options" >&3
              echo "$environment_options"
          }
)


create_toolbox_container_name()
(
    image="$1"

    basename=$(image_reference_get_basename "$image")
    if [ "$basename" = "" ] 2>&3; then
        return 100
    fi

    tag=$(image_reference_get_tag "$image")
    if [ "$tag" = "" ] 2>&3; then
        return 101
    fi

    echo "$basename-$tag"
    return 0
)


create_toolbox_image_name()
(
    # Based on the ResolveName function implemented in:
    # https://github.com/containers/buildah/blob/master/util/util.go

    if image_reference_can_be_id "$base_toolbox_image"; then
        if base_toolbox_image_id=$($podman_command inspect \
                                           --format "{{.Id}}" \
                                           --type image \
                                           "$base_toolbox_image" 2>&3); then
            if has_prefix "$base_toolbox_image_id" "$base_toolbox_image"; then
                echo "$base_toolbox_image-$USER:latest"
                return 0
            fi
        fi
    fi

    basename=$(image_reference_get_basename "$base_toolbox_image")
    if [ "$basename" = "" ] 2>&3; then
        return 100
    fi

    tag=$(image_reference_get_tag "$base_toolbox_image")
    if [ "$tag" = "" ] 2>&3; then
        echo "$basename-$USER:latest"
    else
        echo "$basename-$USER:$tag"
    fi

    return 0
)


enter_print_container_not_found()
(
    container="$1"

    echo "$base_toolbox_command: container $container not found" >&2
    echo "Use the 'create' command to create a toolbox." >&2
    echo "Try '$base_toolbox_command --help' for more information." >&2
)


get_cgroups_version()
(
    version=1

    if ! mounts=$(mount 2>&3); then
        echo "$base_toolbox_command: failed to detect cgroups version: couldn't list mount points" >&2
        return 1
    fi

    if ! (echo "$mounts" | grep "^cgroup " >/dev/null 2>&3) && (echo "$mounts" | grep "^cgroup2 " >/dev/null 2>&3); then
        version=2
    fi

    echo "$version"
    return 0
)


get_group_for_sudo()
(
    group=""
    if getent group sudo >/dev/null 2>&3; then
        group="sudo"
    elif getent group wheel >/dev/null 2>&3; then
        group="wheel"
    else
        return 1
    fi

    echo "$group"
    return 0
)


get_host_id()
(
    # shellcheck disable=SC1091
    . /usr/lib/os-release
    echo "$ID"
)


get_host_variant_id()
(
    # shellcheck disable=SC1091
    . /usr/lib/os-release
    echo "$VARIANT_ID"
)


get_host_version_id()
(
    # shellcheck disable=SC1091
    . /usr/lib/os-release
    echo "$VERSION_ID"
)


image_reference_can_be_id()
(
    image="$1"

    echo "$image" | grep "^[a-f0-9]\{6,64\}$" >/dev/null 2>&3
    return "$?"
)


image_reference_get_basename()
(
    image="$1"

    domain=$(image_reference_get_domain "$image")
    remainder=${image#$domain}
    path=${remainder%:*}
    basename=${path##*/}
    echo "$basename"
)


image_reference_get_domain()
(
    image="$1"

    image_reference_has_domain "$image" && domain=${image%%/*}
    echo "$domain"
)


image_reference_get_tag()
(
    image="$1"

    domain=$(image_reference_get_domain "$image")
    remainder=${image#$domain}

    tag=""
    if (echo "$remainder" | grep ":" >/dev/null 2>&3); then
       tag=${remainder#*:}
    fi

    echo "$tag"
)


image_reference_has_domain()
(
    # Based on the splitDockerDomain function implemented in:
    # https://github.com/docker/distribution/blob/master/reference/normalize.go

    image="$1"

    if ! (echo "$image" | grep "/" >/dev/null 2>&3); then
        return 1
    fi

    prefix=${image%%/*}
    if ! (echo "$prefix" | grep "[.:]" >/dev/null 2>&3) && [ "$prefix" != "localhost" ] 2>&3; then
       return 1
    fi

    return 0
)


images_get_details()
(
    images="$1"

    if ! echo "$images" | while read -r image; do
            [ "$image" = "" ] 2>&3 && continue

            if ! $podman_command images \
                         --format "{{.ID}} {{.Repository}}:{{.Tag}} {{.Created}}" \
                         --noheading \
                         "$image" 2>&3; then
                echo "$base_toolbox_command: failed to get details for image $image" >&2
                return 1
            fi
            echo
         done; then
        return 1
    fi

    return 0
)


is_etc_profile_d_toolbox_a_bind_mount()
{
    container="$1"

    $podman_command inspect --format "[{{range .Mounts}}{{.Dst}} {{end}}]" --type container "$container" 2>&3 \
    | grep /etc/profile.d/toolbox.sh >/dev/null 2>/dev/null 2>&3

    return "$?"
}


list_container_names()
(
    if ! containers_old=$($podman_command ps \
                                  --all \
                                  --filter "label=com.redhat.component=fedora-toolbox" \
                                  --format "{{.Names}}" 2>&3); then
        echo "$base_toolbox_command: failed to list containers with com.redhat.component=fedora-toolbox" >&2
        return 1
    fi

    if ! containers=$($podman_command ps \
                              --all \
                              --filter "label=com.github.debarshiray.toolbox=true" \
                              --format "{{.Names}}" 2>&3); then
        echo "$base_toolbox_command: failed to list containers with com.github.debarshiray.toolbox=true" >&2
        return 1
    fi

    printf "%s\n%s\n" "$containers_old" "$containers" | sort 2>&3 | uniq 2>&3
    return 0
)


mount_bind()
(
    source="$1"
    target="$2"
    mount_flags="$3"

    mount_o=""

    ! [ -d "$source" ] 2>&3 && ! [ -f "$source" ] 2>&3 && return 0

    if [ -d "$source" ] 2>&3; then
        echo "$base_toolbox_command: creating $target" >&3

        if ! mkdir --parents "$target" 2>&3; then
            echo "$base_toolbox_command: failed to create $target" >&2
            return 1
        fi
    fi

    echo "$base_toolbox_command: binding $target to $source" >&3

    [ "$mount_flags" = "" ] 2>&3 || mount_o="-o $mount_flags"

    # shellcheck disable=SC2086
    if ! mount --rbind $mount_o "$source" "$target" 2>&3; then
        echo "$base_toolbox_command: failed to bind $target to $source" >&2
        return 1
    fi

    return 0
)


pull_base_toolbox_image()
(
    domain=""
    has_domain=false
    prompt_for_download=true
    pull_image=false

    if image_reference_can_be_id "$base_toolbox_image"; then
        echo "$base_toolbox_command: looking for image $base_toolbox_image" >&3

        if $podman_command image exists "$base_toolbox_image" >/dev/null 2>&3; then
            return 0
        fi
    fi

    image_reference_has_domain "$base_toolbox_image" && has_domain=true

    if ! $has_domain; then
        echo "$base_toolbox_command: looking for image localhost/$base_toolbox_image" >&3

        if $podman_command image exists localhost/$base_toolbox_image >/dev/null 2>&3; then
            return 0
        fi
    fi

    if $has_domain; then
        base_toolbox_image_full="$base_toolbox_image"
    else
        base_toolbox_image_full="$registry/$fgc/$base_toolbox_image"
    fi

    echo "$base_toolbox_command: looking for image $base_toolbox_image_full" >&3

    if $podman_command image exists "$base_toolbox_image_full" >/dev/null 2>&3; then
        return 0
    fi

    domain=$(image_reference_get_domain "$base_toolbox_image_full")
    if $assume_yes || [ "$domain" = "localhost" ] 2>&3; then
        prompt_for_download=false
        pull_image=true
    fi

    if $prompt_for_download; then
        echo "Image required to create toolbox container."

        prompt=$(printf "Download %s (500MB)? [y/N]:" "$base_toolbox_image_full")
        if ask_for_confirmation "n" "$prompt"; then
            pull_image=true
        else
            pull_image=false
        fi
    fi

    if ! $pull_image; then
        return 1
    fi

    echo "$base_toolbox_command: pulling image $base_toolbox_image_full" >&3

    if spinner_directory=$(mktemp --directory --tmpdir $spinner_template 2>&3); then
        spinner_message="Pulling $base_toolbox_image_full: "
        if ! spinner_start "$spinner_directory" "$spinner_message"; then
            spinner_directory=""
        fi
    else
        echo "$base_toolbox_command: unable to start spinner: spinner directory not created" >&2
        spinner_directory=""
    fi

    $podman_command pull $base_toolbox_image_full >/dev/null 2>&3
    ret_val=$?

    if [ "$spinner_directory" != "" ]; then
        spinner_stop "$spinner_directory"
    fi

    if [ "$ret_val" -ne 0 ] 2>&3; then
        echo "$base_toolbox_command: failed to pull base image $base_toolbox_image" >&2
    fi

    return $ret_val
)


unshare_userns_rm()
(
    path="$1"

    if ! unshare_directory=$(mktemp --directory --tmpdir "toolbox-unshare-userns-rm-XXXXXXXXXX" 2>&3); then
        echo "$base_toolbox_command: failed to enter user namespace: directory couldn't be created" >&2
        return 1
    fi

    if ! touch "$unshare_directory/map" 2>&3; then
        echo "$base_toolbox_command: failed to enter user namespace: file couldn't be created" >&2
        return 1
    fi

    exec 4>"$unshare_directory/map"
    if ! flock 4 2>&3; then
        echo "$base_toolbox_command: failed to enter user namespace: lock couldn't be acquired" >&2
        return 1
    fi

    echo "$base_toolbox_command: parsing /etc/subgid" >&3

    if ! subgid_entry=$(grep "^$USER:" /etc/subgid 2>&3); then
        echo "$base_toolbox_command: failed to enter user namespace: no entry in /etc/subgid" >&2
        return 1
    fi

    userns_gid_start=$(echo "$subgid_entry" | cut --delimiter ":" --fields 2 2>&3)
    if ! is_integer "$userns_gid_start"; then
        echo "$base_toolbox_command: failed to enter user namespace: cannot parse the first sub-GID" >&2
        return 1
    fi

    userns_gid_len=$(echo "$subgid_entry" | cut --delimiter ":" --fields 3 2>&3)
    if ! is_integer "$userns_gid_len"; then
        echo "$base_toolbox_command: failed to enter user namespace: cannot parse the sub-GID count" >&2
        return 1
    fi

    echo "$base_toolbox_command: parsing /etc/subuid" >&3

    if ! subuid_entry=$(grep "^$USER:" /etc/subuid 2>&3); then
        echo "$base_toolbox_command: failed to enter user namespace: no entry in /etc/subuid" >&2
        return 1
    fi

    userns_uid_start=$(echo "$subuid_entry" | cut --delimiter ":" --fields 2 2>&3)
    if ! is_integer "$userns_uid_start"; then
        echo "$base_toolbox_command: failed to enter user namespace: cannot parse the first sub-UID" >&2
        return 1
    fi

    userns_uid_len=$(echo "$subuid_entry" | cut --delimiter ":" --fields 3 2>&3)
    if ! is_integer "$userns_uid_len"; then
        echo "$base_toolbox_command: failed to enter user namespace: cannot parse the sub-UID count" >&2
        return 1
    fi

    echo "$base_toolbox_command: unsharing user namespace" >&3

    unshare --user sh -c "flock $unshare_directory/map rm --force --recursive $path" 2>&3 &
    unshare_pid="$!"

    echo "$base_toolbox_command: setting GID and UID map of user namespace" >&3

    if ! newgidmap "$unshare_pid" 0 "$user_id_real" 1 1 "$userns_gid_start" "$userns_gid_len" 2>&3; then
        echo "$base_toolbox_command: failed to set GID mapping of user namespace" >&2
        kill -9 "$unshare_pid" 2>&3
        return 1
    fi

    if ! newuidmap "$unshare_pid" 0 "$user_id_real" 1 1 "$userns_uid_start" "$userns_uid_len" 2>&3; then
        echo "$base_toolbox_command: failed to set UID mapping of user namespace" >&2
        kill -9 "$unshare_pid" 2>&3
        return 1
    fi

    echo "$base_toolbox_command: GID map of user namespace:" >&3
    cat /proc/"$unshare_pid"/gid_map 1>&3 2>&3

    echo "$base_toolbox_command: UID map of user namespace:" >&3
    cat /proc/"$unshare_pid"/uid_map 1>&3 2>&3

    if ! flock --unlock 4 2>&3; then
        echo "$base_toolbox_command: failed to remove $path: lock couldn't be unlocked" >&2
        kill -9 "$unshare_pid" 2>&3
        return 1
    fi

    if ! wait "$unshare_pid" 2>&3; then
        echo "$base_toolbox_command: failed to remove $path" >&2
        return 1
    fi

    rm --force --recursive "$unshare_directory" 2>&3

    return 0
)


create()
(
    enter_command_skip="$1"

    dbus_system_bus_address="unix:path=/var/run/dbus/system_bus_socket"
    home_link=""
    kcm_socket=""
    kcm_socket_bind=""
    media_link=""
    media_path_bind=""
    mnt_link=""
    mnt_path_bind=""
    run_media_path_bind=""
    toolbox_profile_bind=""
    ulimit_host=""
    usr_mount_destination_flags="ro"

    # shellcheck disable=SC2153
    if [ "$DBUS_SYSTEM_BUS_ADDRESS" != "" ]; then
        dbus_system_bus_address=$DBUS_SYSTEM_BUS_ADDRESS
    fi
    dbus_system_bus_path=$(echo "$dbus_system_bus_address" | cut --delimiter = --fields 2 2>&3)
    dbus_system_bus_path=$(readlink --canonicalize "$dbus_system_bus_path" 2>&3)

    # Note that 'systemctl show ...' doesn't terminate with a non-zero exit
    # code when used with an unknown unit. eg.:
    #   $ systemctl show --value --property Listen foo
    #   $ echo $?
    #   0
    if ! kcm_socket_listen=$(systemctl show --value --property Listen sssd-kcm.socket 2>&3); then
        echo "$base_toolbox_command: failed to use 'systemctl show'" >&3
        kcm_socket_listen=""
    elif [ "$kcm_socket_listen" = "" ] 2>&3; then
        echo "$base_toolbox_command: failed to read property Listen from sssd-kcm.socket" >&3
    else
        echo "$base_toolbox_command: checking value $kcm_socket_listen of property Listen in sssd-kcm.socket" >&3

        if ! (echo "$kcm_socket_listen" | grep " (Stream)$" >/dev/null 2>&3); then
            echo "$base_toolbox_command: unknown socket in sssd-kcm.socket" >&2
            echo "$base_toolbox_command: expected SOCK_STREAM" >&2
            kcm_socket_listen=""
        elif ! (echo "$kcm_socket_listen" | grep "^/" >/dev/null 2>&3); then
            echo "$base_toolbox_command: unknown socket in sssd-kcm.socket" >&2
            echo "$base_toolbox_command: expected file system socket in the AF_UNIX family" >&2
            kcm_socket_listen=""
        fi
    fi

    echo "$base_toolbox_command: parsing value $kcm_socket_listen of property Listen in sssd-kcm.socket" >&3

    if [ "$kcm_socket_listen" != "" ] 2>&3; then
        kcm_socket=${kcm_socket_listen%" (Stream)"}
        kcm_socket_bind="--volume $kcm_socket:$kcm_socket"
    fi

    echo "$base_toolbox_command: checking if 'podman create' supports --ulimit host" >&3

    if man podman-create 2>&3 | grep "You can pass host" >/dev/null 2>&3; then
        echo "$base_toolbox_command: 'podman create' supports --ulimit host" >&3

        ulimit_host="--ulimit host"
    fi

    if ! pull_base_toolbox_image; then
        return 1
    fi

    if image_reference_has_domain "$base_toolbox_image"; then
        base_toolbox_image_full="$base_toolbox_image"
    else
        if ! base_toolbox_image_full=$($podman_command inspect \
                                               --format "{{index .RepoTags 0}}" \
                                               --type image \
                                               "$base_toolbox_image" 2>&3); then
            echo "$base_toolbox_command: failed to get RepoTag for base image $base_toolbox_image" >&2
            return 1
        fi

        echo "$base_toolbox_command: base image $base_toolbox_image resolved to $base_toolbox_image_full" >&3
    fi

    echo "$base_toolbox_command: checking if container $toolbox_container already exists" >&3

    enter_command=$(create_enter_command "$toolbox_container")
    if $podman_command container exists $toolbox_container >/dev/null 2>&3; then
        echo "$base_toolbox_command: container $toolbox_container already exists" >&2
        echo "Enter with: $enter_command" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        return 1
    fi

    if ! group_for_sudo=$(get_group_for_sudo); then
        echo "$base_toolbox_command: failed to create container $toolbox_container: group for sudo not found" >&2
        return 1
    fi

    if [ -f /etc/profile.d/toolbox.sh ] 2>&3; then
        toolbox_profile_bind="--volume /etc/profile.d/toolbox.sh:/etc/profile.d/toolbox.sh:ro"
    elif [ -f /usr/share/profile.d/toolbox.sh ] 2>&3; then
        toolbox_profile_bind="--volume /usr/share/profile.d/toolbox.sh:/etc/profile.d/toolbox.sh:ro"
    else
        echo "$base_toolbox_command: failed to bind mount toolbox.sh" >&3
    fi

    if [ -d /media ] 2>&3; then
        echo "$base_toolbox_command: checking if /media is a symbolic link to /run/media" >&3

        if [ "$(readlink /media)" = run/media ] 2>&3; then
            echo "$base_toolbox_command: /media is a symbolic link to /run/media" >&3
            media_link="--media-link"
        else
            media_path_bind="--volume /media:/media:rslave"
        fi
    fi

    echo "$base_toolbox_command: checking if /mnt is a symbolic link to /var/mnt" >&3

    if [ "$(readlink /mnt)" = var/mnt ] 2>&3; then
        echo "$base_toolbox_command: /mnt is a symbolic link to /var/mnt" >&3
        mnt_link="--mnt-link"
    else
        mnt_path_bind="--volume /mnt:/mnt:rslave"
    fi

    if [ -d /run/media ] 2>&3; then
        run_media_path_bind="--volume /run/media:/run/media:rslave"
    fi

    echo "$base_toolbox_command: checking if /usr is mounted read-only or read-write" >&3

    if ! usr_mount_point=$(df --output=target /usr | tail --lines 1 2>&3); then
        echo "$base_toolbox_command: failed to get the mount-point of /usr" >&2
    else
        echo "$base_toolbox_command: mount-point of /usr is $usr_mount_point" >&3

        if ! usr_mount_source_flags=$(findmnt --noheadings --output OPTIONS "$usr_mount_point" 2>&3); then
            echo "$base_toolbox_command: failed to get the mount options of $usr_mount_point" >&2
        else
            echo "$base_toolbox_command: mount flags of /usr on the host are $usr_mount_source_flags" >&3

            if echo "$usr_mount_source_flags" | grep --invert-match "ro" >/dev/null 2>&3; then
                usr_mount_destination_flags="rw"
            fi
        fi
    fi

    if ! home_canonical=$(readlink --canonicalize "$HOME" 2>&3); then
        echo "$base_toolbox_command: failed to canonicalize $HOME" >&2
        return 1
    fi

    echo "$base_toolbox_command: $HOME canonicalized to $home_canonical" >&3

    echo "$base_toolbox_command: checking if /home is a symbolic link to /var/home" >&3

    if [ "$(readlink /home)" = var/home ] 2>&3; then
	echo "$base_toolbox_command: /home is a symbolic link to /var/home" >&3
	home_link="--home-link"
    fi

    echo "$base_toolbox_command: calling org.freedesktop.Flatpak.SessionHelper.RequestSession" >&3

    if ! gdbus call \
                 --session \
                 --dest org.freedesktop.Flatpak \
                 --object-path /org/freedesktop/Flatpak/SessionHelper \
                 --method org.freedesktop.Flatpak.SessionHelper.RequestSession >/dev/null 2>&3; then
        echo "$base_toolbox_command: failed to call org.freedesktop.Flatpak.SessionHelper.RequestSession" >&2
        exit 1
    fi

    echo "$base_toolbox_command: creating container $toolbox_container" >&3

    if spinner_directory=$(mktemp --directory --tmpdir $spinner_template 2>&3); then
        spinner_message="Creating container $toolbox_container: "
        if ! spinner_start "$spinner_directory" "$spinner_message"; then
            spinner_directory=""
        fi
    else
        echo "$base_toolbox_command: unable to start spinner: spinner directory not created" >&2
        spinner_directory=""
    fi

    # shellcheck disable=SC2086
    $podman_command create \
            --dns none \
            --env TOOLBOX_PATH="$TOOLBOX_PATH" \
            --group-add "$group_for_sudo" \
            --hostname toolbox \
            --ipc host \
            --label "com.github.containers.toolbox=true" \
            --label "com.github.debarshiray.toolbox=true" \
            --name $toolbox_container \
            --network host \
            --no-hosts \
            --pid host \
            --privileged \
            --security-opt label=disable \
            $ulimit_host \
            --userns=keep-id \
            --user root:root \
            $kcm_socket_bind \
            $media_path_bind \
            $mnt_path_bind \
            $run_media_path_bind \
            $toolbox_profile_bind \
            --volume "$TOOLBOX_PATH":/usr/bin/toolbox:ro \
            --volume "$XDG_RUNTIME_DIR":"$XDG_RUNTIME_DIR" \
            --volume "$XDG_RUNTIME_DIR"/.flatpak-helper/monitor:/run/host/monitor \
            --volume "$dbus_system_bus_path":"$dbus_system_bus_path" \
            --volume "$home_canonical":"$home_canonical":rslave \
            --volume /etc:/run/host/etc \
            --volume /dev:/dev:rslave \
            --volume /run:/run/host/run:rslave \
            --volume /tmp:/run/host/tmp:rslave \
            --volume /usr:/run/host/usr:"$usr_mount_destination_flags",rslave \
            --volume /var:/run/host/var:rslave \
            "$base_toolbox_image_full" \
            toolbox --verbose init-container \
                    --home "$HOME" \
                    $home_link \
                    $media_link \
                    $mnt_link \
                    --monitor-host \
                    --shell "$SHELL" \
                    --uid "$user_id_real" \
                    --user "$USER" >/dev/null 2>&3
    ret_val=$?

    if [ "$spinner_directory" != "" ]; then
        spinner_stop "$spinner_directory"
    fi

    if [ $ret_val -ne 0 ]; then
        echo "$base_toolbox_command: failed to create container $toolbox_container" >&2
        return 1
    fi

    if ! $enter_command_skip; then
        echo "Created container: $toolbox_container"
        echo "Enter with: $enter_command"
    fi

    return 0
)


enter()
(
    emit_escape_sequence=false
    host_id=$(get_host_id)
    host_variant_id=$(get_host_variant_id)

    if [ "$host_id" = "fedora" ] 2>&3 \
       && { [ "$host_variant_id" = "silverblue" ] 2>&3 || [ "$host_variant_id" = "workstation" ] 2>&3; }; then
        emit_escape_sequence=true
    fi

    run "$emit_escape_sequence" true false "$SHELL" -l
)


init_container()
{
    init_container_home="$1"
    init_container_home_link="$2"
    init_container_media_link="$3"
    init_container_mnt_link="$4"
    init_container_monitor_host="$5"
    init_container_shell="$6"
    init_container_uid="$7"
    init_container_user="$8"

    if [ "$XDG_RUNTIME_DIR" = "" ] 2>&3; then
        echo "$base_toolbox_command: XDG_RUNTIME_DIR is unset" >&3

        XDG_RUNTIME_DIR=/run/user/"$init_container_uid"
        echo "$base_toolbox_command: XDG_RUNTIME_DIR set to $XDG_RUNTIME_DIR" >&3

        toolbox_runtime_directory="$XDG_RUNTIME_DIR"/toolbox
    fi

    init_container_initialized_stamp="$toolbox_runtime_directory"/container-initialized-"$$"

    echo "$base_toolbox_command: creating /run/.toolboxenv" >&3

    if ! touch /run/.toolboxenv 2>&3; then
        echo "$base_toolbox_command: failed to create /run/.toolboxenv" >&2
        return 1
    fi

    if $init_container_monitor_host; then
        working_directory="$PWD"

        if [ -d /run/host/etc ] 2>&3; then
            if ! readlink /etc/host.conf >/dev/null 2>&3; then
                echo "$base_toolbox_command: redirecting /etc/host.conf to /run/host/etc/host.conf" >&3

                if ! (cd /etc 2>&3 \
                      && rm --force host.conf 2>&3 \
                      && ln --symbolic /run/host/etc/host.conf host.conf 2>&3); then
                    echo "$base_toolbox_command: failed to redirect /etc/host.conf to /run/host/etc/host.conf" >&2
                    return 1
                fi
            fi

            if ! readlink /etc/hosts >/dev/null 2>&3; then
                echo "$base_toolbox_command: redirecting /etc/hosts to /run/host/etc/hosts" >&3

                if ! (cd /etc 2>&3 \
                      && rm --force hosts 2>&3 \
                      && ln --symbolic /run/host/etc/hosts hosts 2>&3); then
                    echo "$base_toolbox_command: failed to redirect /etc/hosts to /run/host/etc/hosts" >&2
                    return 1
                fi
            fi

            if ! readlink /etc/resolv.conf >/dev/null 2>&3; then
                echo "$base_toolbox_command: redirecting /etc/resolv.conf to /run/host/etc/resolv.conf" >&3

                if ! (cd /etc 2>&3 \
                      && rm --force resolv.conf 2>&3 \
                      && ln --symbolic /run/host/etc/resolv.conf resolv.conf 2>&3); then
                    echo "$base_toolbox_command: failed to redirect /etc/resolv.conf to /run/host/etc/resolv.conf" \
                            >&2
                    return 1
                fi
            fi


            if ! mount_bind /run/host/etc/machine-id /etc/machine-id ro; then
                return 1
            fi

            if ! mount_bind /run/host/run/libvirt /run/libvirt; then
                return 1
            fi

            if ! mount_bind /run/host/run/systemd/journal /run/systemd/journal; then
                return 1
            fi

            if [ -d /sys/fs/selinux ] 2>&3; then
                if ! mount_bind /usr/share/empty /sys/fs/selinux; then
                    return 1
                fi
            fi

            if ! mount_bind /run/host/var/lib/flatpak /var/lib/flatpak ro; then
                return 1
            fi

            if ! mount_bind /run/host/var/log/journal /var/log/journal ro; then
                return 1
            fi

            if ! mount_bind /run/host/var/mnt /var/mnt rslave; then
                return 1
            fi
        fi

        if [ -d /run/host/monitor ] 2>&3; then
            if ! localtime_target=$(readlink /etc/localtime >/dev/null 2>&3) \
               || [ "$localtime_target" != "/run/host/monitor/localtime" ] 2>&3; then
                echo "$base_toolbox_command: redirecting /etc/localtime to /run/host/monitor/localtime" >&3

                if ! (cd /etc 2>&3 \
                      && rm --force localtime 2>&3 \
                      && ln --symbolic /run/host/monitor/localtime localtime 2>&3); then
                    echo "$base_toolbox_command: failed to redirect /etc/localtime to /run/host/monitor/localtime" \
                            >&2
                    return 1
                fi
            fi

            if ! readlink /etc/timezone >/dev/null 2>&3; then
                echo "$base_toolbox_command: redirecting /etc/timezone to /run/host/monitor/timezone" >&3

                if ! (cd /etc 2>&3 \
                      && rm --force timezone 2>&3 \
                      && ln --symbolic /run/host/monitor/timezone timezone 2>&3); then
                    echo "$base_toolbox_command: failed to redirect /etc/timezone to /run/host/monitor/timezone" >&2
                    return 1
                fi
            fi
        fi

        if ! cd "$working_directory" 2>&3; then
            echo "$base_toolbox_command: failed to restore working directory" >&2
        fi
    fi

    if $init_container_media_link && ! readlink /media >/dev/null 2>&3; then
        echo "$base_toolbox_command: making /media a symbolic link to /run/media" >&3

        # shellcheck disable=SC2174
        if ! (rmdir /media 2>&3 \
              && mkdir --mode 0755 --parents /run/media 2>&3 \
              && ln --symbolic run/media /media 2>&3); then
            echo "$base_toolbox_command: failed to make /media a symbolic link" >&2
            return 1
        fi
    fi

    if $init_container_mnt_link && ! readlink /mnt >/dev/null 2>&3; then
        echo "$base_toolbox_command: making /mnt a symbolic link to /var/mnt" >&3

        # shellcheck disable=SC2174
        if ! (rmdir /mnt 2>&3 \
              && mkdir --mode 0755 --parents /var/mnt 2>&3 \
              && ln --symbolic var/mnt /mnt 2>&3); then
            echo "$base_toolbox_command: failed to make /mnt a symbolic link" >&2
            return 1
        fi
    fi

    if ! id -u "$init_container_user" >/dev/null 2>&3; then
        if $init_container_home_link ; then
            echo "$base_toolbox_command: making /home a symlink" >&3

            # shellcheck disable=SC2174
            if ! (rmdir /home 2>&3 \
                  && mkdir --mode 0755 --parents /var/home 2>&3 \
                  && ln --symbolic var/home /home 2>&3); then
                echo "$base_toolbox_command: failed to make /home a symlink" >&2
                return 1
            fi
        fi

        if ! groups=$(get_group_for_sudo); then
            echo "$base_toolbox_command: failed to add user $init_container_user: group for sudo not found" >&2
            return 1
        fi

        echo "$base_toolbox_command: adding user $init_container_user with UID $init_container_uid" >&3

        if ! useradd \
                     --home-dir "$init_container_home" \
                     --no-create-home \
                     --shell "$init_container_shell" \
                     --uid "$init_container_uid" \
                     --groups "$groups" \
                     "$init_container_user" >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to add user $init_container_user with UID $init_container_uid" >&2
            return 1
        fi

        echo "$base_toolbox_command: removing password for user $init_container_user" >&3

        if ! passwd --delete "$init_container_user" >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove password for user $init_container_user" >&2
            return 1
        fi

        echo "$base_toolbox_command: removing password for user root" >&3

        if ! passwd --delete root >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove password for user root" >&2
            return 1
        fi

    fi

    if [ -d /etc/krb5.conf.d ] 2>&3 && ! [ -f /etc/krb5.conf.d/kcm_default_ccache ] 2>&3; then
        echo "$base_toolbox_command: setting KCM as the default Kerberos credential cache" >&3

        cat <<EOF >/etc/krb5.conf.d/kcm_default_ccache 2>&3
# Written by Toolbox
# https://github.com/debarshiray/toolbox
#
# # To disable the KCM credential cache, comment out the following lines.

[libdefaults]
    default_ccache_name = KCM:
EOF
        ret_val=$?

        if [ "$ret_val" -ne 0 ] 2>&3; then
            echo "$base_toolbox_command: failed to set KCM as the default Kerberos credential cache" >&2
            return 1
        fi
    fi

    echo "$base_toolbox_command: finished initializing container" >&3

    if ! touch "$init_container_initialized_stamp" 2>&3; then
        echo "$base_toolbox_command: failed to create initialization stamp" >&2
        return 1
    fi

    echo "$base_toolbox_command: going to sleep" >&3

    exec sleep +Inf
}


run()
(
    emit_escape_sequence="$1"
    fallback_to_bash="$2"
    pedantic="$3"
    program="$4"
    shift 4

    create_toolbox_container=false
    prompt_for_create=true

    echo "$base_toolbox_command: checking if container $toolbox_container exists" >&3

    if ! $podman_command container exists "$toolbox_container" 2>&3; then
        echo "$base_toolbox_command: container $toolbox_container not found" >&3

        if $podman_command container exists "$toolbox_container_old_v1" 2>&3; then
            echo "$base_toolbox_command: container $toolbox_container_old_v1 found" >&3

            # shellcheck disable=SC2030
            toolbox_container="$toolbox_container_old_v1"
        elif $podman_command container exists "$toolbox_container_old_v2" 2>&3; then
            echo "$base_toolbox_command: container $toolbox_container_old_v2 found" >&3

            # shellcheck disable=SC2030
            toolbox_container="$toolbox_container_old_v2"
        else
            if $pedantic; then
                enter_print_container_not_found "$toolbox_container"
                exit 1
            fi

            if ! containers=$(list_container_names); then
                enter_print_container_not_found "$toolbox_container"
                exit 1
            fi

            containers_count=$(echo "$containers" | grep --count . 2>&3)
            if ! is_integer "$containers_count"; then
                enter_print_container_not_found "$toolbox_container"
                exit 1
            fi

            echo "$base_toolbox_command: found $containers_count containers" >&3

            if [ "$containers_count" -eq 0 ] 2>&3; then
                if $assume_yes; then
                    create_toolbox_container=true
                    prompt_for_create=false
                fi

                if $prompt_for_create; then
                    prompt="No toolbox containers found. Create now? [y/N]"
                    if ask_for_confirmation "n" "$prompt"; then
                        create_toolbox_container=true
                    else
                        create_toolbox_container=false
                    fi
                fi

                if ! $create_toolbox_container; then
                    echo "A container can be created later with the 'create' command." >&2
                    echo "Try '$base_toolbox_command --help' for more information." >&2
                    exit 1
                fi

                if ! update_container_and_image_names; then
                    exit 1
                fi

                if ! create true; then
                    exit 1
                fi
            elif [ "$containers_count" -eq 1 ] 2>&3 \
                 && [ "$toolbox_container" = "$toolbox_container_default" ] 2>&3; then
                echo "$base_toolbox_command: container $toolbox_container not found" >&2

                toolbox_container=$(echo "$containers" | grep . 2>&3 | head --lines 1 2>&3)
                echo "Entering container $toolbox_container instead." >&2
                echo "Use the 'create' command to create a different toolbox." >&2
                echo "Try '$base_toolbox_command --help' for more information." >&2
            else
                echo "$base_toolbox_command: container $toolbox_container not found" >&2
                echo "Use the '--container' option to select a toolbox." >&2
                echo "Try '$base_toolbox_command --help' for more information." >&2
                exit 1
            fi
        fi
    fi

    echo "$base_toolbox_command: calling org.freedesktop.Flatpak.SessionHelper.RequestSession" >&3

    if ! gdbus call \
                 --session \
                 --dest org.freedesktop.Flatpak \
                 --object-path /org/freedesktop/Flatpak/SessionHelper \
                 --method org.freedesktop.Flatpak.SessionHelper.RequestSession >/dev/null 2>&3; then
        echo "$base_toolbox_command: failed to call org.freedesktop.Flatpak.SessionHelper.RequestSession" >&2
        exit 1
    fi

    echo "$base_toolbox_command: starting container $toolbox_container" >&3

    if is_etc_profile_d_toolbox_a_bind_mount "$toolbox_container"; then
        echo "$base_toolbox_command: /etc/profile.d/toolbox.sh already mounted in container $toolbox_container" >&3

        if ! container_start "$toolbox_container"; then
            exit 1
        fi
    else
        echo "$base_toolbox_command: /etc/profile.d/toolbox.sh not mounted in container $toolbox_container" >&3

        if ! copy_etc_profile_d_toolbox_to_runtime_directory; then
            exit 1
        fi

        if ! container_start "$toolbox_container"; then
            exit 1
        fi

        if ! copy_etc_profile_d_toolbox_to_container "$toolbox_container"; then
            exit 1
        fi
    fi

    echo "$base_toolbox_command: inspecting entry point of container $toolbox_container" >&3

    if ! entry_point=$($podman_command inspect --format "{{index .Config.Cmd 0}}" --type container "$toolbox_container" 2>&3); then
        echo "$base_toolbox_command: failed to inspect entry point of container $toolbox_container" >&2
        exit 1
    fi

    echo "$base_toolbox_command: entry point of container $toolbox_container is $entry_point" >&3

    if [ "$entry_point" = "toolbox" ] 2>&3; then
        echo "$base_toolbox_command: waiting for container $toolbox_container to finish initializing" >&3

        if ! entry_point_pid=$($podman_command inspect --format "{{.State.Pid}}" --type container "$toolbox_container" 2>&3); then
            echo "$base_toolbox_command: failed to inspect entry point PID of container $toolbox_container" >&2
            exit 1
        fi

        if ! is_integer "$entry_point_pid"; then
            echo "$base_toolbox_command: failed to parse entry point PID of container $toolbox_container" >&2
            exit 1
        fi

        if [ "$entry_point_pid" -le 0 ] 2>&3; then
            echo "$base_toolbox_command: invalid entry point PID of container $toolbox_container" >&2
            exit 1
        fi

        container_initialized_stamp="$toolbox_runtime_directory/container-initialized-$entry_point_pid"
        container_initialized_timeout=25 #s

        i=0
        while ! [ -f "$container_initialized_stamp" ] 2>&3; do
            sleep 1 2>&3

            i=$((i + 1))
            if [ "$i" -eq "$container_initialized_timeout" ] 2>&3; then
                echo "$base_toolbox_command: failed to initialize container $toolbox_container" >&2
                exit 1
            fi
        done
    else
        echo "$base_toolbox_command: container $toolbox_container uses deprecated features" >&2
        echo "Consider recreating it with Toolbox version 0.0.17 or newer." >&2
    fi

    if ! $podman_command exec --user root:root "$toolbox_container" touch /run/.toolboxenv 2>&3; then
        echo "$base_toolbox_command: failed to create /run/.toolboxenv in container $toolbox_container" >&2
        exit 1
    fi

    set_environment=$(create_environment_options)

    echo "$base_toolbox_command: looking for $program in container $toolbox_container" >&3

    # shellcheck disable=SC2016
    if ! $podman_command exec \
                 --user "$USER" \
                 "$toolbox_container" \
                 sh -c 'command -v "$1"' sh "$program" >/dev/null 2>&3; then
        if $fallback_to_bash; then
            echo "$base_toolbox_command: $program not found in $toolbox_container; using /bin/bash instead" >&3
            program=/bin/bash
        else
            echo "$base_toolbox_command: command '$program' not found in container $toolbox_container" >&2
            exit 127
        fi
    fi

    if ! $podman_command exec \
                 --user "$USER" \
                 "$toolbox_container" \
                 sh -c "test -d $PWD" >/dev/null 2>&3; then
        echo "Directory $PWD does not exist in container $toolbox_container, try cd && !!" >&2
        exit 127
    fi

    echo "$base_toolbox_command: running in container $toolbox_container:" >&3
    echo "$base_toolbox_command: $program" >&3
    for i in "$@"; do
        echo "$base_toolbox_command: $i" >&3
    done

    $emit_escape_sequence && printf "\033]777;container;push;%s;toolbox\033\\" "$toolbox_container"

    tty_arguments="--interactive --tty"
    if $notty; then
	tty_arguments=""
    fi

    # shellcheck disable=SC2016
    # for the command passed to capsh
    # shellcheck disable=SC2086
    $podman_command exec \
            $tty_arguments \
            --user "$USER" \
            --workdir "$PWD" \
            $set_environment \
            "$toolbox_container" \
            capsh --caps="" -- -c 'exec "$@"' /bin/sh "$program" "$@" 2>&3
    ret_val="$?"

    $emit_escape_sequence && printf "\033]777;container;pop;;\033\\"

    exit "$ret_val"
)


help()
(
    to_help_command="$1"

    if [ "$to_help_command" = "" ] 2>&3 || [ "$to_help_command" = "$base_toolbox_command" ] 2>&3; then
        exec man toolbox 2>&1
    fi

    exec man toolbox-"$to_help_command" 2>&1
)


list_images()
(
    output=""

    if ! images_old=$($podman_command images \
                              --filter "label=com.redhat.component=fedora-toolbox" \
                              --format "{{.Repository}}:{{.Tag}}" 2>&3); then
        echo "$base_toolbox_command: failed to list images with com.redhat.component=fedora-toolbox" >&2
        return 1
    fi

    if ! images=$($podman_command images \
                          --filter "label=com.github.debarshiray.toolbox=true" \
                          --format "{{.Repository}}:{{.Tag}}" 2>&3); then
        echo "$base_toolbox_command: failed to list images with com.github.debarshiray.toolbox=true" >&2
        return 1
    fi

    images=$(printf "%s\n%s\n" "$images_old" "$images" | sort 2>&3 | uniq 2>&3)
    if ! details=$(images_get_details "$images"); then
        return 1
    fi

    if [ "$details" != "" ] 2>&3; then
        table_data=$(printf "%s\t%s\t%s\n" "IMAGE ID" "IMAGE NAME" "CREATED"; echo "$details")
        if ! output=$(echo "$table_data" | sed "s/ \{2,\}/\t/g" 2>&3 | column -s "$tab" -t 2>&3); then
            echo "$base_toolbox_command: failed to parse list of images" >&2
            return 1
        fi
    fi

    if [ "$output" != "" ]; then
        echo "$output"
    fi

    return 0
)


containers_get_details()
(
    containers="$1"

    if ! echo "$containers" | while read -r container; do
            [ "$container" = "" ] 2>&3 && continue

            if ! $podman_command ps --all \
                         --filter "name=$container" \
                         --format "{{.ID}}  {{.Names}}  {{.Created}}  {{.Status}}  {{.Image}}" 2>&3; then
                echo "$base_toolbox_command: failed to get details for container $container" >&2
                return 1
            fi
         done; then
        return 1
    fi

    return 0
)


list_containers()
(
    output=""

    if ! containers=$(list_container_names); then
        return 1
    fi

    if ! details=$(containers_get_details "$containers"); then
        return 1
    fi

    if [ "$details" != "" ] 2>&3; then
        table_data=$(printf "%s\t%s\t%s\t%s\t%s\n" "CONTAINER ID" "CONTAINER NAME" "CREATED" "STATUS" "IMAGE NAME"
                     echo "$details")
        if ! output=$(echo "$table_data" | sed "s/ \{2,\}/\t/g" 2>&3 | column -s "$tab" -t 2>&3); then
            echo "$base_toolbox_command: failed to parse list of containers" >&2
            return 1
        fi
    fi

    if [ "$output" != "" ]; then
        echo "$output" | head --lines 1 2>&3

        echo "$output" | tail --lines +2 2>&3 \
            | (
                  while read -r container; do
                      id=$(echo "$container" | cut --delimiter " " --fields 1 2>&3)
                      is_running=$($podman_command inspect "$id" --format "{{.State.Running}}" 2>&3)
                      if $is_running; then
                          # shellcheck disable=SC2059
                          printf "${LGC}$container${NC}\n"
                      else
                          echo "$container"
                      fi
                  done
              )
    fi

    return 0
)


migrate()
(
    configuration_directory="$HOME/.config/toolbox"
    migrate_stamp="$configuration_directory/podman-system-migrate"

    migrate_lock="$toolbox_runtime_directory"/migrate.lock

    if ! version=$($podman_command version --format "{{.Version}}" 2>&3); then
        echo "$base_toolbox_command: unable to migrate containers: Podman version couldn't be read" >&2
        return 1
    fi

    echo "$base_toolbox_command: current Podman version is $version" >&3

    if ! mkdir --parents "$configuration_directory" 2>&3; then
        echo "$base_toolbox_command: unable to migrate containers: configuration directory not created" >&2
        return 1
    fi

    # shellcheck disable=SC2174
    if ! mkdir --mode 700 --parents "$toolbox_runtime_directory" 2>&3; then
        echo "$base_toolbox_command: unable to migrate containers: runtime directory not created" >&2
        return 1
    fi

    exec 5>"$migrate_lock"
    if ! flock 5 2>&3; then
        echo "$base_toolbox_command: unable to migrate containers: migration lock not acquired" >&3
        return 1
    fi

    if [ -f "$migrate_stamp" ] 2>&3; then
        if grep "$version" "$migrate_stamp" >/dev/null 2>&3; then
            echo "$base_toolbox_command: migration not needed: Podman version $version is unchanged" >&3
            return 0
        fi

        if ! version_old=$(printf "%s\n" "$version" \
                           | cat "$migrate_stamp" - 2>&3 \
                           | sort --version-sort 2>&3 \
                           | head --lines 1 2>&3); then
            echo "$base_toolbox_command: unable to migrate containers: Podman versions couldn't be sorted" >&2
            return 1
        fi

        if [ "$version" = "$version_old" ] 2>&3; then
            echo "$base_toolbox_command: migration not needed: Podman version $version is old" >&3
            return 0
        fi
    fi

    if ! $podman_command system migrate >/dev/null 2>&3; then
        echo "$base_toolbox_command: unable to migrate containers" >&2
        return 1
    fi

    echo "$version" >"$migrate_stamp"
    return 0
)


remove_containers()
(
    ids=$1
    all=$2
    force=$3

    ret_val=0

    $force && force_option="--force"

    if $all; then
        if ! ids_old=$($podman_command ps \
                               --all \
                               --filter "label=com.redhat.component=fedora-toolbox" \
                               --format "{{.ID}}" 2>&3); then
            echo "$base_toolbox_command: failed to list containers with com.redhat.component=fedora-toolbox" >&2
            return 1
        fi

        if ! ids=$($podman_command ps \
                           --all \
                           --filter "label=com.github.debarshiray.toolbox=true" \
                           --format "{{.ID}}" 2>&3); then
            echo "$base_toolbox_command: failed to list containers with com.github.debarshiray.toolbox=true" >&2
            return 1
        fi

        ids=$(printf "%s\n%s\n" "$ids_old" "$ids" | sort 2>&3 | uniq 2>&3)
        if [ "$ids" != "" ]; then
            ret_val=$(echo "$ids" \
                      | (
                            while read -r id; do
                                if ! $podman_command rm $force_option "$id" >/dev/null 2>&3; then
                                    echo "$base_toolbox_command: failed to remove container $id" >&2
                                    ret_val=1
                                fi
                            done

                            echo "$ret_val"
                        )
                     )
        fi
    else
        ret_val=$(echo "$ids" \
                  | sed "s/ \+/\n/g" 2>&3 \
                  | (
                        while read -r id; do
                            if ! labels=$($podman_command inspect \
                                                  --format "{{.Config.Labels}}" \
                                                  --type container \
                                                  "$id" 2>&3); then
                                echo "$base_toolbox_command: failed to inspect $id" >&2
                                ret_val=1
                                continue
                            fi

                            if ! has_substring "$labels" "com.github.debarshiray.toolbox" \
                               && ! has_substring "$labels" "com.redhat.component:fedora-toolbox"; then
                                echo "$base_toolbox_command: $id is not a toolbox container" >&2
                                ret_val=1
                                continue
                            fi

                            if ! $podman_command rm $force_option "$id" >/dev/null 2>&3; then
                                echo "$base_toolbox_command: failed to remove container $id" >&2
                                ret_val=1
                            fi
                        done

                        echo "$ret_val"
                    )
                 )
    fi

    return "$ret_val"
)


remove_images()
(
    ids=$1
    all=$2
    force=$3

    ret_val=0

    $force && force_option="--force"

    if $all; then
        if ! ids_old=$($podman_command images \
                               --filter "label=com.redhat.component=fedora-toolbox" \
                               --format "{{.ID}}" 2>&3); then
            echo "$0: failed to list images with com.redhat.component=fedora-toolbox" >&2
            return 1
        fi

        if ! ids=$($podman_command images \
                           --all \
                           --filter "label=com.github.debarshiray.toolbox=true" \
                           --format "{{.ID}}" 2>&3); then
            echo "$0: failed to list images with com.github.debarshiray.toolbox=true" >&2
            return 1
        fi

        ids=$(printf "%s\n%s\n" "$ids_old" "$ids" | sort 2>&3 | uniq 2>&3)
        if [ "$ids" != "" ]; then
            ret_val=$(echo "$ids" \
                      | (
                            while read -r id; do
                                if ! $podman_command rmi $force_option "$id" >/dev/null 2>&3; then
                                    echo "$base_toolbox_command: failed to remove image $id" >&2
                                    ret_val=1
                                fi
                            done

                            echo "$ret_val"
                        )
                     )
        fi
    else
        ret_val=$(echo "$ids" \
                  | sed "s/ \+/\n/g" 2>&3 \
                  | (
                        while read -r id; do
                            if ! labels=$($podman_command inspect \
                                                  --format "{{.Labels}}" \
                                                  --type image \
                                                  "$id" 2>&3); then
                                echo "$base_toolbox_command: failed to inspect $id" >&2
                                ret_val=1
                                continue
                            fi

                            if ! has_substring "$labels" "com.github.debarshiray.toolbox" \
                               && ! has_substring "$labels" "com.redhat.component:fedora-toolbox"; then
                                echo "$base_toolbox_command: $id is not a toolbox image" >&2
                                ret_val=1
                                continue
                            fi

                            if ! $podman_command rmi $force_option "$id" >/dev/null 2>&3; then
                                echo "$base_toolbox_command: failed to remove image $id" >&2
                                ret_val=1
                            fi
                        done

                        echo "$ret_val"
                    )
                 )
    fi

    return "$ret_val"
)


reset()
(
    do_reset=false
    prompt_for_reset=true
    ret_val=0

    if [ "$user_id_real" -eq 0 ] 2>&3; then
        if [ -d /run/containers ] 2>&3; then
            echo "$base_toolbox_command: The 'reset' command cannot be used after other commands" >&2
            echo "Reboot the system before using it again." >&2
            echo "Try '$base_toolbox_command --help' for more information." >&2
            return 1
        fi
    else
        if [ -d "$XDG_RUNTIME_DIR"/overlay-containers ] 2>&3 \
           || [ -d "$XDG_RUNTIME_DIR"/overlay-layers ] 2>&3 \
           || [ -d "$XDG_RUNTIME_DIR"/overlay-locks ] 2>&3; then
            echo "$base_toolbox_command: The 'reset' command cannot be used after other commands" >&2
            echo "Reboot the system before using it again." >&2
            echo "Try '$base_toolbox_command --help' for more information." >&2
            return 1
        fi
    fi

    if $assume_yes; then
        do_reset=true
        prompt_for_reset=false
    fi

    if $prompt_for_reset; then
        echo "All existing podman (and toolbox) containers and images will be removed."

        prompt=$(printf "Continue? [y/N]:")
        if ask_for_confirmation "n" "$prompt"; then
            do_reset=true
        else
            do_reset=false
        fi
    fi

    if ! $do_reset; then
        return 1
    fi

    echo "$base_toolbox_command: resetting local state" >&3

    if [ "$user_id_real" -eq 0 ] 2>&3; then
        if ! rm --force --recursive /var/lib/containers/cache >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove /var/lib/containers/cache" >&2
            ret_val=1
        fi

        if ! rm --force --recursive /var/lib/containers/sigstore/* >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove the contents of /var/lib/containers/sigstore" >&2
            ret_val=1
        fi

        if ! rm --force --recursive /var/lib/containers/storage >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove /var/lib/containers/storage" >&2
            ret_val=1
        fi
    else
        if ! unshare_userns_rm "$HOME/.local/share/containers"; then
            ret_val=1
        fi

        if ! rm --force --recursive "$HOME/.config/containers" >/dev/null 2>&3; then
            echo "$base_toolbox_command: failed to remove $HOME/.config/containers" >&2
            ret_val=1
        fi
    fi

    if ! rm --force --recursive "$HOME/.config/toolbox" >/dev/null 2>&3; then
        echo "$base_toolbox_command: failed to remove $HOME/.config/toolbox" >&2
        ret_val=1
    fi

    return "$ret_val"
)


exit_if_extra_operand()
{
    if [ "$1" != "" ]; then
        echo "$base_toolbox_command: extra operand '$1'" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
    fi
}


exit_if_missing_argument()
{
    if [ "$2" = "" ]; then
        echo "$base_toolbox_command: missing argument for '$1'" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
    fi
}


exit_if_non_positive_argument()
{
    if ! is_integer "$2"; then
        echo "$base_toolbox_command: invalid argument for '$1'" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
    fi
    if [ "$2" -le 0 ] 2>&3; then
        echo "$base_toolbox_command: invalid argument for '$1'" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
    fi
}


exit_if_unrecognized_option()
{
    echo "$base_toolbox_command: unrecognized option '$1'" >&2
    echo "Try '$base_toolbox_command --help' for more information." >&2
    exit 1
}


# shellcheck disable=SC2120
forward_to_host()
(
    if ! command -v flatpak-spawn >/dev/null 2>&3; then
        echo "$base_toolbox_command: flatpak-spawn not found" >&2
        return 1
    fi

    eval "set -- $arguments"

    set_environment=$(create_environment_options)

    echo "$base_toolbox_command: forwarding to host:" >&3
    echo "$base_toolbox_command: $TOOLBOX_PATH" >&3
    for i in "$@"; do
        echo "$base_toolbox_command: $i" >&3
    done

    # shellcheck disable=SC2086
    flatpak-spawn $set_environment --host "$TOOLBOX_PATH" "$@" 2>&3
    ret_val="$?"

    return "$ret_val"
)


update_container_and_image_names()
{
    [ "$release" = "" ] 2>&3 && release="$release_default"

    if [ "$base_toolbox_image" = "" ] 2>&3; then
        base_toolbox_image="fedora-toolbox:$release"
    else
        release=$(image_reference_get_tag "$base_toolbox_image")
        [ "$release" = "" ] 2>&3 && release="$release_default"
    fi

    fgc="f$release"
    echo "$base_toolbox_command: Fedora generational core is $fgc" >&3

    echo "$base_toolbox_command: base image is $base_toolbox_image" >&3

    toolbox_image=$(create_toolbox_image_name)
    if ! (
            ret_val=$?
            if [ "$ret_val" -ne 0 ] 2>&3; then
                if [ "$ret_val" -eq 100 ] 2>&3; then
                    echo "$base_toolbox_command: failed to get the basename of base image $base_toolbox_image" >&2
                else
                    echo "$base_toolbox_command: failed to create an ID for the customized user-specific image" >&2
                fi

                exit 1
            fi

            exit 0
         ); then
        return 1
    fi

    # shellcheck disable=SC2031
    if [ "$toolbox_container" = "" ]; then
        toolbox_container=$(create_toolbox_container_name "$base_toolbox_image")
        if ! (
                 ret_val=$?
                 if [ "$ret_val" -ne 0 ] 2>&3; then
                     if [ "$ret_val" -eq 100 ] 2>&3; then
                         echo "$base_toolbox_command: failed to get the basename of image $base_toolbox_image" >&2
                     elif [ "$ret_val" -eq 101 ] 2>&3; then
                         echo "$base_toolbox_command: failed to get the tag of image $base_toolbox_image" >&2
                     else
                         echo "$base_toolbox_command: failed to create a name for the toolbox container" >&2
                     fi

                     exit 1
                 fi

                 exit 0
            ); then
            return 1
        fi

        if ! container_name_is_valid "$toolbox_container"; then
            echo "$base_toolbox_command: generated container name $toolbox_container is invalid" >&2
            echo "Container names must match '$container_name_regexp'." >&2
            echo "Try '$base_toolbox_command --help' for more information." >&2
            return 1
        fi

        toolbox_container_old_v1=$(create_toolbox_container_name "$toolbox_image")
        if ! (
                 ret_val=$?
                 if [ "$ret_val" -ne 0 ] 2>&3; then
                     if [ "$ret_val" -eq 100 ] 2>&3; then
                         echo "$base_toolbox_command: failed to get the basename of image $toolbox_image" >&2
                     elif [ "$ret_val" -eq 101 ] 2>&3; then
                         echo "$base_toolbox_command: failed to get the tag of image $toolbox_image" >&2
                     else
                         echo "$base_toolbox_command: failed to create a name for the toolbox container" >&2
                     fi

                     exit 1
                 fi

                 exit 0
            ); then
            return 1
        fi

        toolbox_container_old_v2="$toolbox_image"
    fi

    echo "$base_toolbox_command: container is $toolbox_container" >&3
    return 0
}

arguments=$(save_positional_parameters "$@")

host_id=$(get_host_id)
if [ "$host_id" = "fedora" ] 2>&3; then
    release_default=$(get_host_version_id)
else
    release_default="30"
fi
toolbox_container_prefix_default="fedora-toolbox"
toolbox_container_default="$toolbox_container_prefix_default-$release_default"

while has_prefix "$1" -; do
    case $1 in
        --assumeyes | -y )
            assume_yes=true
            ;;
        -h | --help )
            if [ -f /run/.containerenv ] 2>&3; then
                if ! [ -f /run/.toolboxenv ] 2>&3; then
                    echo "$base_toolbox_command: this is not a toolbox container" >&2
                    exit 1
                fi

                # shellcheck disable=SC2119
                forward_to_host
                exit
            fi

            help "$2"
            exit
            ;;
        -v | --verbose )
            exec 3>&2
            verbose=true
            ;;
        -vv | --very-verbose )
            exec 3>&2
            podman_command="podman --log-level debug"
            verbose=true
            ;;
        * )
            exit_if_unrecognized_option "$1"
    esac
    shift
done

echo "$base_toolbox_command: running as real user ID $user_id_real" >&3

if ! toolbox_command_path=$(realpath "$0" 2>&3); then
    echo "$base_toolbox_command: failed to resolve absolute path to $0" >&2
    exit 1
fi

echo "$base_toolbox_command: resolved absolute path for $0 to $toolbox_command_path" >&3

if [ -f /run/.containerenv ] 2>&3; then
    if [ "$TOOLBOX_PATH" = "" ] 2>&3; then
        echo "$base_toolbox_command: TOOLBOX_PATH not set" >&2
        exit 1
    fi
else
    if [ "$user_id_real" -ne 0 ] 2>&3; then
        echo "$base_toolbox_command: checking if /etc/subgid and /etc/subuid have entries for user $USER" >&3

        if ! grep "^$USER:" /etc/subgid >/dev/null 2>&3 || ! grep "^$USER:" /etc/subuid >/dev/null 2>&3; then
            echo "$base_toolbox_command: /etc/subgid and /etc/subuid don't have entries for user $USER" >&2
            echo "See the podman(1), subgid(5), subuid(5) and usermod(8) manuals for more" >&2
            echo "information." >&2
            exit 1
        fi
    fi

    if [ "$TOOLBOX_PATH" = "" ] 2>&3; then
        TOOLBOX_PATH="$toolbox_command_path"
    fi
fi

echo "$base_toolbox_command: TOOLBOX_PATH is $TOOLBOX_PATH" >&3

if [ "$1" = "" ]; then
    echo "$base_toolbox_command: missing command" >&2
    echo >&2
    echo "These are some common commands:" >&2
    echo "create    Create a new toolbox container" >&2
    echo "enter     Enter an existing toolbox container" >&2
    echo "list      List all existing toolbox containers and images" >&2
    echo >&2
    echo "Try '$base_toolbox_command --help' for more information." >&2
    exit 1
fi

op=$1
shift

if [ -f /run/.containerenv ] 2>&3; then
    case $op in
        create | enter | list | rm | rmi | run | help )
            if ! [ -f /run/.toolboxenv ] 2>&3; then
                echo "$base_toolbox_command: this is not a toolbox container" >&2
                exit 1
            fi

            # shellcheck disable=SC2119
            forward_to_host
            exit "$?"
            ;;
        init-container )
            init_container_home_link=false
            init_container_media_link=false
            init_container_mnt_link=false
            init_container_monitor_host=false
            while has_prefix "$1" -; do
                case $1 in
                    -h | --help )
                        # shellcheck disable=SC2119
                        forward_to_host
                        exit
                        ;;
                    --home )
                        shift
                        exit_if_missing_argument --home "$1"
                        init_container_home="$1"
                        ;;
                    --home-link )
                        init_container_home_link=true
                        ;;
                    --media-link )
                        init_container_media_link=true
                        ;;
                    --mnt-link )
                        init_container_mnt_link=true
                        ;;
                    --monitor-host )
                        init_container_monitor_host=true
                        ;;
                    --shell )
                        shift
                        exit_if_missing_argument --shell "$1"
                        init_container_shell="$1"
                        ;;
                    --uid )
                        shift
                        exit_if_missing_argument --uid "$1"
                        init_container_uid="$1"
                        ;;
                    --user )
                        shift
                        exit_if_missing_argument --user "$1"
                        init_container_user="$1"
                        ;;
                    * )
                        exit_if_unrecognized_option "$1"
                esac
                shift
            done
            init_container \
                    "$init_container_home" \
                    "$init_container_home_link" \
                    "$init_container_media_link" \
                    "$init_container_mnt_link" \
                    "$init_container_monitor_host" \
                    "$init_container_shell" \
                    "$init_container_uid" \
                    "$init_container_user"
            exit "$?"
            ;;
        reset )
            echo "$base_toolbox_command: The 'reset' command cannot be used inside containers" >&2
            echo "Try '$base_toolbox_command --help' for more information." >&2
            exit 1
            ;;
        * )
           echo "$base_toolbox_command: unrecognized command '$op'" >&2
           echo "Try '$base_toolbox_command --help' for more information." >&2
           exit 1
           ;;
    esac
fi

if ! cgroups_version=$(get_cgroups_version); then
    exit 1
fi

echo "$base_toolbox_command: running on a cgroups v$cgroups_version host" >&3

if [ "$op" != "reset" ] 2>&3; then
    if ! migrate; then
        exit 1
    fi
fi

case $op in
    create )
        while has_prefix "$1" -; do
            case $1 in
                --candidate-registry )
                    registry=$registry_candidate
                    ;;
                -c | --container )
                    shift
                    exit_if_missing_argument --container "$1"
                    arg=$1
                    if ! container_name_is_valid "$arg"; then
                        echo "$base_toolbox_command: invalid argument for '--container'" >&2
                        echo "Container names must match '$container_name_regexp'." >&2
                        echo "Try '$base_toolbox_command --help' for more information." >&2
                        exit 1
                    fi
                    toolbox_container="$arg"
                    ;;
                -h | --help )
                    help "$op"
                    exit
                    ;;
                -i | --image )
                    shift
                    exit_if_missing_argument --image "$1"
                    base_toolbox_image=$1
                    ;;
                -r | --release )
                    shift
                    exit_if_missing_argument --release "$1"
                    arg=$(echo "$1" | sed "s/^F\|^f//" 2>&3)
                    exit_if_non_positive_argument --release "$arg"
                    release=$arg
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        exit_if_extra_operand "$1"
        if ! update_container_and_image_names; then
            exit 1
        fi
        if ! create false; then
            exit 1
        fi
        exit
        ;;
    enter )
        while has_prefix "$1" -; do
            case $1 in
                -c | --container )
                    shift
                    exit_if_missing_argument --container "$1"
                    toolbox_container=$1
                    ;;
                -h | --help )
                    help "$op"
                    exit
                    ;;
                -r | --release )
                    shift
                    exit_if_missing_argument --release "$1"
                    arg=$(echo "$1" | sed "s/^F\|^f//" 2>&3)
                    exit_if_non_positive_argument --release "$arg"
                    release=$arg
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        exit_if_extra_operand "$1"
        if ! update_container_and_image_names; then
            exit 1
        fi
        enter
        exit
        ;;
    help )
        while has_prefix "$1" -; do
            case $1 in
                -h | --help )
                    help "$op"
                    exit
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        help "$1"
        exit
        ;;
    init-container )
        while has_prefix "$1" -; do
            case $1 in
                -h | --help )
                    help "$op"
                    exit
            esac
            shift
        done
        echo "$base_toolbox_command: The 'init-container' command can only be used inside containers" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
        ;;
    list )
        ls_add_empty_line=false
        ls_images=false
        ls_containers=false
        while has_prefix "$1" -; do
            case $1 in
                -c | --containers )
                    ls_containers=true
                    ;;
                -h | --help )
                    help "$op"
                    exit
                    ;;
                -i | --images )
                    ls_images=true
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        exit_if_extra_operand "$1"

        if ! $ls_containers && ! $ls_images; then
            ls_containers=true
            ls_images=true
        fi

        if $ls_images; then
            if ! images=$(list_images); then
                exit 1
            fi
        fi

        if $ls_containers; then
            if ! containers=$(list_containers); then
                exit 1
            fi
        fi

        if $ls_images && [ "$images" != "" ] 2>&3; then
            echo "$images"
            ls_add_empty_line=true
        fi

        if $ls_containers && [ "$containers" != "" ] 2>&3; then
            $ls_add_empty_line && echo ""
            echo "$containers"
        fi

        exit
        ;;
    reset )
        while has_prefix "$1" -; do
            case $1 in
                -h | --help )
                    help "$op"
                    exit
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        exit_if_extra_operand "$1"

        reset
        exit "$?"
        ;;
    rm | rmi )
        rm_all=false
        rm_force=false
        while has_prefix "$1" -; do
            case $1 in
                -a | --all )
                    rm_all=true
                    ;;
                -f | --force )
                    rm_force=true
                    ;;
                -h | --help )
                    help "$op"
                    exit
                    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done

        rm_ids=""
        if $rm_all; then
            exit_if_extra_operand "$1"
        else
            exit_if_missing_argument "$op" "$1"
            while [ "$1" != "" ]; do
                rm_ids="$rm_ids $1"
                shift
            done
        fi

        rm_ids=$(echo "$rm_ids" | sed "s/^ \+//" 2>&3)

        if [ "$op" = "rm" ]; then
            remove_containers "$rm_ids" "$rm_all" "$rm_force"
        else
            remove_images "$rm_ids" "$rm_all" "$rm_force"
        fi
        exit
        ;;
    run )
        while has_prefix "$1" -; do
            case $1 in
                -c | --container )
                    shift
                    exit_if_missing_argument --container "$1"
                    toolbox_container=$1
                    ;;
                -h | --help )
                    help "$op"
                    exit
                    ;;
                -r | --release )
                    shift
                    exit_if_missing_argument --release "$1"
                    arg=$(echo "$1" | sed "s/^F\|^f//" 2>&3)
                    exit_if_non_positive_argument --release "$arg"
                    release=$arg
                    ;;
		-n | --notty )
		    notty=true
		    ;;
                * )
                    exit_if_unrecognized_option "$1"
            esac
            shift
        done
        exit_if_missing_argument "$op" "$1"
        if ! update_container_and_image_names; then
            exit 1
        fi
        run false false true "$@"
        exit
        ;;
    * )
        echo "$base_toolbox_command: unrecognized command '$op'" >&2
        echo "Try '$base_toolbox_command --help' for more information." >&2
        exit 1
esac

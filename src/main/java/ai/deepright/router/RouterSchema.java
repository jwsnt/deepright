package ai.deepright.router;

import com.fasterxml.jackson.annotation.JsonPropertyDescription;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class RouterSchema {

    @JsonPropertyDescription("The member description, including purpose and functions")
    protected String description;

    @JsonPropertyDescription("The working directory of the device")
    protected String workspace;

    @JsonPropertyDescription("The terminal execution environment of the device (e.g., zsh)")
    protected String terminal;

    @JsonPropertyDescription("The same gateway MAC address (e.g., 9c:a6:15:5:3f:ba) indicates that the devices are on the same network")
    // arp -n $(route -n get default | grep gateway | awk '{print $2}') | awk '{print $4}'
    protected String gateway;

    // 设备编号
    @JsonPropertyDescription("The member device unique ID")
    protected String device;

    @JsonPropertyDescription("The member agent unique ID")
    protected String agent;

    @JsonPropertyDescription("The data directory of the device")
    protected String dir;

    @JsonPropertyDescription("The operating system of the device (e.g., Darwin 23.4.0)")
    protected String sys;
}

<app:application xmlns:app="http://www.sierrawireless.com/airvantage/application/1.0" type="SWIR_M3DA_WATCHMAN" name="M3da Watchman Test Application" revision="1">
  <capabilities>
    <communication>
      <protocol comm-id="SERIAL" type="M3DA">
        <parameter name="authentication" value="HMAC-SHA1"/>
        <parameter name="cipher" value="none"/>
      </protocol>
    </communication>
    <data>
      <encoding type="M3DA">
        <asset default-label="Basic Asset" id="@sys.secure.telemetry">
          <setting default-label="Temperature" path="temperature" />
          <setting default-label="Humidity" path="humidity" />
          <setting default-label="Pressure" path="pressure" />

          <setting default-label="Location" path="location" />
          <setting default-label="Security Level" path="security_level" />
        </asset>

        <asset default-label="Compressed Asset" id="@sys.compressed.telemetry">
          <setting default-label="Temperature" path="temperature" />
          <setting default-label="Unit" path="unit" />
          <setting default-label="Sensor ID" path="sensor_id" />

          <setting default-label="Encrypted" path="encrypted" />
          <setting default-label="Compression Format" path="compression" />
        </asset>

        <asset id="get_secure_status">
        <command path="get_secure_status" default-label="Slam the door" >
          <parameter id="target" type="string" optional="true" ></parameter>
          <parameter id="security_token" type="string" optional="true" ></parameter>
          <parameter id="timestamp" type="int" optional="true" ></parameter>
        </command>
        </asset>
      </encoding>
    </data>
  </capabilities>
</app:application>
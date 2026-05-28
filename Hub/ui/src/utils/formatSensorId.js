/**
 * Strip the hw-sensor- prefix for display purposes.
 * Raw sensorId is always used for logic (keys, API calls, tooltips).
 *
 * @param {string} sensorId
 * @returns {string}
 */
export const formatSensorId = (sensorId) => {
    if (!sensorId) return sensorId
    return sensorId.replace(/^hw-sensor-/, '')
}
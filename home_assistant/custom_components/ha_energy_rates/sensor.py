from homeassistant.components.sensor import SensorEntity
from homeassistant.helpers.update_coordinator import CoordinatorEntity
from .const import DOMAIN

def _get_path(data, path):
    cur = data
    for part in path.split("."):
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
        if cur is None:
            return None
    return cur

async def async_setup_entry(hass, entry, async_add_entities):
    coord = hass.data[DOMAIN][entry.entry_id]
    provider = entry.data["provider"]

    sensors = [
        FieldSensor(coord, entry,
            "rates.residential_standard.energy_rate_usd_per_kwh",
            f"{provider} Energy Rate", "USD/kWh", "energy_rate"),
        FieldSensor(coord, entry,
            "rates.residential_standard.tva_fuel_rate_usd_per_kwh",
            f"{provider} Fuel Rate", "USD/kWh", "fuel_rate"),
        TotalRateSensor(coord, entry,
            "rates.residential_standard.energy_rate_usd_per_kwh",
            "rates.residential_standard.tva_fuel_rate_usd_per_kwh",
            f"{provider} Total Rate", "USD/kWh", "total_rate"),
        FieldSensor(coord, entry,
            "rates.residential_standard.customer_charge_monthly_usd",
            f"{provider} Fixed Charge", "USD", "fixed_charge"),
    ]

    async_add_entities(sensors)


class BaseEnergySensor(CoordinatorEntity, SensorEntity):
    def __init__(self, coordinator, entry, name, unit, key):
        super().__init__(coordinator)
        self._attr_name = name
        self._attr_unique_id = f"{entry.entry_id}_{key}"
        self._attr_native_unit_of_measurement = unit
        self._attr_device_class = "monetary"


class FieldSensor(BaseEnergySensor):
    def __init__(self, coordinator, entry, path, name, unit, key):
        self._path = path
        super().__init__(coordinator, entry, name, unit, key)

    @property
    def native_value(self):
        data = self.coordinator.data or {}
        v = _get_path(data, self._path)
        if v is None:
            return None
        try:
            return float(v)
        except Exception:
            return None


class TotalRateSensor(BaseEnergySensor):
    def __init__(self, coordinator, entry, ekey, fkey, name, unit, key):
        self._ekey = ekey
        self._fkey = fkey
        super().__init__(coordinator, entry, name, unit, key)

    @property
    def native_value(self):
        data = self.coordinator.data or {}
        e = _get_path(data, self._ekey) or 0
        f = _get_path(data, self._fkey) or 0
        try:
            return float(e) + float(f)
        except Exception:
            return None

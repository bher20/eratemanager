from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant

from .const import DOMAIN
from .coordinator import EnergyRatesCoordinator

async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry):
    coord = EnergyRatesCoordinator(hass, entry)
    await coord.async_config_entry_first_refresh()
    hass.data.setdefault(DOMAIN, {})[entry.entry_id] = coord
    await hass.config_entries.async_forward_entry_setups(entry, ["sensor"])

    async def _register():
        provider = entry.data["provider"]
        sensor_entity = f"sensor.{provider}_total_rate"
        try:
            await hass.services.async_call(
                "energy",
                "register_price",
                {"stat_cost": sensor_entity},
                blocking=True,
            )
        except Exception:
            # Energy dashboard not set up yet or service missing.
            pass

    hass.async_create_task(_register())
    return True

async def async_unload_entry(hass, entry: ConfigEntry):
    await hass.config_entries.async_forward_entry_unload(entry, "sensor")
    hass.data[DOMAIN].pop(entry.entry_id, None)
    return True

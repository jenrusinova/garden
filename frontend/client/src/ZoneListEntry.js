

var ZoneListEntry = ({zone}) => {
  console.log(zone);
  var zoneName = zone.name;
  var isWorking = zone.is_on.toString();
  var runTime = zone.runtime;
  return (
    <div className ='zone-list-entry'>
    <h3><div className = 'zone-list-entry-name'>
     {zoneName}
    </div></h3>
    <div className = 'zone-list-entry-working'>
      Active: <button>{isWorking}</button>
    </div>
    <div className = 'zone-list-entry-runtime'>
      Time:{runTime}
    </div>
    </div>
  )



}

export default ZoneListEntry;
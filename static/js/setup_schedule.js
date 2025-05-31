// /static/js/setup_schedule.js
// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side methods for the schedule-generation page.
// Now supports entering “Matches per Team” and displays the computed cycle time.

var blockTemplate   = Handlebars.compile($("#blockTemplate").html());
var blockMatches    = {};   // blockNumber → numMatches in that block
var blockCycleTime  = {};   // blockNumber → cycle time (in seconds)

//
// Adds a new scheduling block to the page. 
// If both arguments are omitted, it creates a default block. Otherwise,
// `startTime` (a Moment) and `matchesPerTeam` (integer) are used to pre-populate.
//
var addBlock = function(startTime, matchesPerTeam) {
  var lastBlockNumber = getLastBlockNumber();

  // If no explicit startTime was provided, generate defaults:
  if (!startTime) {
    if ($.isEmptyObject(blockMatches)) {
      // First block on an empty page:
      matchesPerTeam = 1;                   // default: 1 match per team
      var defaultCycleSec = 360;            // default 6-minute cycle
      startTime = moment().add(1, "hour").startOf("hour");
      var numMatches = Math.ceil(matchesPerTeam * numTeams / 6);
      var endTime    = moment(startTime).add(numMatches * defaultCycleSec, "seconds");
    } else {
      // Subsequent block: start at the previous block’s actual end time,
      // reuse that block’s matchesPerTeam and cycle time:
      var prevNum = lastBlockNumber;
      var prevStartRaw = moment($("#startTime" + prevNum).val(), "YYYY-MM-DD hh:mm:ss A");
      var prevEndRaw   = moment($("#endTime"   + prevNum).val(), "YYYY-MM-DD hh:mm:ss A");
      var prevActualEnd = moment($("#actualEndTime" + prevNum).text(), "hh:mm:ss A");
      
      startTime = prevActualEnd.clone();
      matchesPerTeam = parseInt($("#matchesPerTeam" + prevNum).val(), 10);

      var prevTotalSec = prevEndRaw.diff(prevStartRaw) / 1000;
      var prevMatches  = blockMatches[prevNum] || 0;
      var prevCycleSec = (prevMatches > 0 ? Math.floor(prevTotalSec / prevMatches) : 360);

      var numMatches = Math.ceil(matchesPerTeam * numTeams / 6);
      var endTime    = moment(startTime).add(numMatches * prevCycleSec, "seconds");
    }
  }

  // If `startTime` exists but `endTime` wasn’t set above, use defaults:
  if (startTime && typeof endTime === "undefined") {
    matchesPerTeam = matchesPerTeam || 1;
    var defaultCycleSec = 360;
    var numMatches = Math.ceil(matchesPerTeam * numTeams / 6);
    var endTime    = moment(startTime).add(numMatches * defaultCycleSec, "seconds");
  }

  // Create the new block in the DOM:
  lastBlockNumber += 1;
  var blockHtml = blockTemplate({ blockNumber: lastBlockNumber });
  $("#blockContainer").append(blockHtml);

  // Initialize date/time pickers:
  newDateTimePicker("startTimePicker" + lastBlockNumber, startTime.toDate());
  newDateTimePicker("endTimePicker"   + lastBlockNumber, endTime.toDate());

  // Whenever the “Matches per Team” input or date/time pickers change, recalc:
  $("#matchesPerTeam" + lastBlockNumber).on("input", function() {
    updateBlock(lastBlockNumber);
  });
  $("#startTime" + lastBlockNumber).on("change", function() {
    updateBlock(lastBlockNumber);
  });
  $("#endTime" + lastBlockNumber).on("change", function() {
    updateBlock(lastBlockNumber);
  });

  // Perform the initial calculation for this block:
  updateBlock(lastBlockNumber);
};


//
// Recomputes “numMatches”, “cycleTime”, and “actualEndTime” for a block.
//
var updateBlock = function(blockNumber) {
  var startTime = moment($("#startTime" + blockNumber).val(), "YYYY-MM-DD hh:mm:ss A");
  var endTime   = moment($("#endTime"   + blockNumber).val(), "YYYY-MM-DD hh:mm:ss A");

  var rawMPT = $("#matchesPerTeam" + blockNumber).val();
  var mpt    = parseInt(rawMPT, 10);

  if (!isNaN(mpt) && mpt > 0 && startTime.isValid() && endTime.isValid() && endTime.isAfter(startTime)) {
    // (1) Total matches in this block = ceil(matchesPerTeam * numTeams / 6)
    var numMatches = Math.ceil(mpt * numTeams / 6);

    // (2) Compute cycle time (in seconds) so that exactly numMatches fit
    var totalSec    = endTime.diff(startTime) / 1000;
    var cycleSec    = Math.floor(totalSec / numMatches);

    // (3) Recompute actual end time based on cycleSec:
    var actualEndMoment = startTime.clone().add(numMatches * cycleSec, "seconds");
    var actualEnd      = actualEndMoment.format("hh:mm:ss A");

    // (4) Format cycleSec as “m:ss”
    var minutes = Math.floor(cycleSec / 60);
    var seconds = cycleSec % 60;
    var formattedCycle = minutes + ":" + (seconds < 10 ? "0" + seconds : seconds);

    // Update the DOM:
    $("#cycleTime"      + blockNumber).text(formattedCycle);
    $("#numMatches"     + blockNumber).text(numMatches);
    $("#actualEndTime"  + blockNumber).text(actualEnd);

    // Store for later (e.g. when generating the form):
    blockMatches[blockNumber]   = numMatches;
    blockCycleTime[blockNumber] = cycleSec;
  } else {
    // Invalid or incomplete inputs → clear fields:
    $("#cycleTime"      + blockNumber).text("");
    $("#numMatches"     + blockNumber).text("");
    $("#actualEndTime"  + blockNumber).text("");

    blockMatches[blockNumber]   = 0;
    blockCycleTime[blockNumber] = 0;
  }

  // Update global statistics:
  updateStats();
};


//
// Recomputes and displays global stats (totalNumMatches, matchesPerTeam, excess, next-level).
//
var updateStats = function() {
  var totalNumMatches = 0;
  $.each(blockMatches, function(k, v) {
    totalNumMatches += v;
  });

  // Global “matches per team” = floor(totalMatches * 6 / numTeams)
  var globalMPT        = Math.floor(totalNumMatches * 6 / numTeams);
  var numExcessMatches = totalNumMatches - Math.ceil(globalMPT * numTeams / 6);
  var nextLevelMatches = Math.ceil((globalMPT + 1) * numTeams / 6) - totalNumMatches;

  $("#totalNumMatches").text(totalNumMatches);
  $("#matchesPerTeam"  ).text(globalMPT);
  $("#numExcessMatches").text(numExcessMatches);
  $("#nextLevelMatches").text(nextLevelMatches);
};


//
// Removes a block from the page and updates global stats.
//
var deleteBlock = function(blockNumber) {
  delete blockMatches[blockNumber];
  delete blockCycleTime[blockNumber];
  $("#block" + blockNumber).remove();
  updateStats();
};


//
// Builds a POST form with hidden fields capturing each block’s data, then submits.
//
var generateSchedule = function() {
  var form = $("#scheduleForm");
  form.attr("method", "POST");
  form.attr("action", "/setup/schedule/generate");

  var addField = function(name, value) {
    var field = $(document.createElement("input"));
    field.attr("type", "hidden");
    field.attr("name", name);
    field.attr("value", value);
    form.append(field);
  };

  var i = 0;
  $.each(blockMatches, function(k, v) {
    addField("startTime"        + i, $("#startTime"        + k).val());
    addField("numMatches"       + i, blockMatches[k]);
    addField("matchSpacingSec"  + i, blockCycleTime[k]);
    addField("matchesPerTeam"   + i, $("#matchesPerTeam"   + k).val());
    i++;
  });
  addField("numScheduleBlocks", i);
  form.submit();
};


//
// Returns the highest numeric key among existing blocks (0 if none).
//
var getLastBlockNumber = function() {
  var max = 0;
  $.each(blockMatches, function(k, v) {
    var num = parseInt(k, 10);
    if (!isNaN(num) && num > max) {
      max = num;
    }
  });
  return max;
};
